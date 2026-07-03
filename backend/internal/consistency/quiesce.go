package consistency

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// QuiesceState represents the state of a quiesced database.
type QuiesceState string

const (
	StateNormal    QuiesceState = "normal"
	StateQuiescing QuiesceState = "quiescing"
	StateQuiesced  QuiesceState = "quiesced"
	StateResuming  QuiesceState = "resuming"
	StateFailed    QuiesceState = "failed"
)

// QuiesceOperation represents a database quiesce operation.
type QuiesceOperation struct {
	DatabaseType string
	DatabaseName string
	State        QuiesceState
	StartTime    time.Time
	QuiesceTime  time.Time
	ResumeTime   time.Time
	Error        error
	Metadata     map[string]interface{}
}

// QuiesceManager manages database quiesce operations.
type QuiesceManager struct {
	mu         sync.RWMutex
	operations map[string]*QuiesceOperation
}

// NewQuiesceManager creates a new quiesce manager.
func NewQuiesceManager() *QuiesceManager {
	return &QuiesceManager{
		operations: make(map[string]*QuiesceOperation),
	}
}

// PostgreSQLQuiescer handles PostgreSQL quiesce operations.
type PostgreSQLQuiescer struct {
	db            *sql.DB
	conn          *sql.Conn // pinned connection for the non-exclusive backup session
	backupLabel   string
	checkpointLSN string
}

// NewPostgreSQLQuiescer creates a new PostgreSQL quiescer.
func NewPostgreSQLQuiescer(connString string) (*PostgreSQLQuiescer, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return &PostgreSQLQuiescer{
		db:          db,
		backupLabel: fmt.Sprintf("backup_%d", time.Now().Unix()),
	}, nil
}

// Quiesce starts a PostgreSQL backup session.
func (pq *PostgreSQLQuiescer) Quiesce(ctx context.Context) (*QuiesceOperation, error) {
	op := &QuiesceOperation{
		DatabaseType: "postgresql",
		State:        StateQuiescing,
		StartTime:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Pin a dedicated connection. A non-exclusive pg_start_backup session is
	// tied to the backend that started it: pg_stop_backup MUST be issued on the
	// SAME physical connection, otherwise the backup is never finalized. Using
	// the pooled *sql.DB would let stop land on a different backend and silently
	// fail to honor the session.
	conn, err := pq.db.Conn(ctx)
	if err != nil {
		op.State = StateFailed
		op.Error = fmt.Errorf("failed to acquire pinned connection: %w", err)
		return op, op.Error
	}

	// Execute pg_start_backup
	// This function ensures that the database is in a consistent state for backup
	var lsn string
	query := "SELECT pg_start_backup($1, false, false)"
	err = conn.QueryRowContext(ctx, query, pq.backupLabel).Scan(&lsn)
	if err != nil {
		_ = conn.Close()
		op.State = StateFailed
		op.Error = fmt.Errorf("pg_start_backup failed: %w", err)
		return op, op.Error
	}

	pq.conn = conn
	pq.checkpointLSN = lsn
	op.QuiesceTime = time.Now()
	op.State = StateQuiesced
	op.Metadata["backup_label"] = pq.backupLabel
	op.Metadata["checkpoint_lsn"] = lsn
	op.Metadata["quiesce_duration_ms"] = op.QuiesceTime.Sub(op.StartTime).Milliseconds()

	// Get current database size (same pinned connection)
	var dbSize int64
	sizeQuery := "SELECT pg_database_size(current_database())"
	if err := conn.QueryRowContext(ctx, sizeQuery).Scan(&dbSize); err == nil {
		op.Metadata["database_size_bytes"] = dbSize
	}

	// Get WAL information (same pinned connection)
	var walFile string
	walQuery := "SELECT pg_walfile_name(pg_current_wal_lsn())"
	if err := conn.QueryRowContext(ctx, walQuery).Scan(&walFile); err == nil {
		op.Metadata["current_wal_file"] = walFile
	}

	return op, nil
}

// Resume ends the PostgreSQL backup session on the pinned connection.
func (pq *PostgreSQLQuiescer) Resume(ctx context.Context) error {
	if pq.conn == nil {
		return nil
	}

	// Execute pg_stop_backup on the SAME connection that started the backup,
	// then release the pinned connection back to the pool.
	var lsn string
	query := "SELECT pg_stop_backup(false, true)"
	err := pq.conn.QueryRowContext(ctx, query).Scan(&lsn)
	closeErr := pq.conn.Close()
	pq.conn = nil
	if err != nil {
		return fmt.Errorf("pg_stop_backup failed: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to release pinned connection: %w", closeErr)
	}

	return nil
}

// Close closes the PostgreSQL connection.
func (pq *PostgreSQLQuiescer) Close() error {
	// Ensure the backup session is ended and the pinned connection released.
	var resumeErr error
	if pq.conn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resumeErr = pq.Resume(ctx)
	}
	if err := pq.db.Close(); err != nil {
		return err
	}
	return resumeErr
}

// MySQLQuiescer handles MySQL quiesce operations.
type MySQLQuiescer struct {
	db               *sql.DB
	conn             *sql.Conn // pinned connection holding the read lock
	tablesLocked     bool
	lockTimeout      time.Duration
	originalReadOnly bool
}

// NewMySQLQuiescer creates a new MySQL quiescer.
func NewMySQLQuiescer(connString string, lockTimeout time.Duration) (*MySQLQuiescer, error) {
	db, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	if lockTimeout == 0 {
		lockTimeout = 30 * time.Second
	}

	return &MySQLQuiescer{
		db:          db,
		lockTimeout: lockTimeout,
	}, nil
}

// Quiesce locks all MySQL tables for consistent backup.
func (mq *MySQLQuiescer) Quiesce(ctx context.Context) (*QuiesceOperation, error) {
	op := &QuiesceOperation{
		DatabaseType: "mysql",
		State:        StateQuiescing,
		StartTime:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Pin a dedicated connection. FLUSH TABLES WITH READ LOCK is session-scoped:
	// the lock is held only by the connection that issued it and is released by
	// UNLOCK TABLES (or disconnect) on that SAME connection. Issuing lock/unlock
	// over the pooled *sql.DB could land UNLOCK on a different physical
	// connection, leaving the database silently frozen.
	conn, err := mq.db.Conn(ctx)
	if err != nil {
		op.State = StateFailed
		op.Error = fmt.Errorf("failed to acquire pinned connection: %w", err)
		return op, op.Error
	}

	// Check current read_only status
	var readOnlyVar, readOnlyValue string
	if rerr := conn.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'read_only'").Scan(&readOnlyVar, &readOnlyValue); rerr == nil {
		mq.originalReadOnly = (readOnlyValue == "ON")
		op.Metadata["original_read_only"] = mq.originalReadOnly
	}

	// Set lock wait timeout on the pinned connection
	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET SESSION lock_wait_timeout = %d", int(mq.lockTimeout.Seconds())))
	if err != nil {
		_ = conn.Close()
		op.State = StateFailed
		op.Error = fmt.Errorf("failed to set lock timeout: %w", err)
		return op, op.Error
	}

	// Execute FLUSH TABLES WITH READ LOCK on the pinned connection
	// This ensures all tables are flushed to disk and locked for reading
	_, err = conn.ExecContext(ctx, "FLUSH TABLES WITH READ LOCK")
	if err != nil {
		_ = conn.Close()
		op.State = StateFailed
		op.Error = fmt.Errorf("FLUSH TABLES WITH READ LOCK failed: %w", err)
		return op, op.Error
	}

	mq.conn = conn
	mq.tablesLocked = true
	op.QuiesceTime = time.Now()
	op.State = StateQuiesced
	op.Metadata["lock_duration_ms"] = op.QuiesceTime.Sub(op.StartTime).Milliseconds()

	// Get binary log position (same pinned connection)
	var binlogFile string
	var binlogPos int64
	err = conn.QueryRowContext(ctx, "SHOW MASTER STATUS").Scan(&binlogFile, &binlogPos)
	if err == nil {
		op.Metadata["binlog_file"] = binlogFile
		op.Metadata["binlog_position"] = binlogPos
	}

	// Get database size (same pinned connection)
	var dbSize sql.NullInt64
	sizeQuery := "SELECT SUM(data_length + index_length) FROM information_schema.tables WHERE table_schema = DATABASE()"
	if err := conn.QueryRowContext(ctx, sizeQuery).Scan(&dbSize); err == nil && dbSize.Valid {
		op.Metadata["database_size_bytes"] = dbSize.Int64
	}

	// Get table count (same pinned connection)
	var tableCount int
	countQuery := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()"
	if err := conn.QueryRowContext(ctx, countQuery).Scan(&tableCount); err == nil {
		op.Metadata["table_count"] = tableCount
	}

	return op, nil
}

// Resume unlocks all MySQL tables on the pinned connection.
func (mq *MySQLQuiescer) Resume(ctx context.Context) error {
	if !mq.tablesLocked {
		return nil
	}

	if mq.conn == nil {
		// Lock flag set without a pinned connection should not happen; treat the
		// lock as released since the holding connection is gone.
		mq.tablesLocked = false
		return nil
	}

	// Execute UNLOCK TABLES on the SAME connection that holds the read lock,
	// then release the pinned connection back to the pool.
	_, err := mq.conn.ExecContext(ctx, "UNLOCK TABLES")
	closeErr := mq.conn.Close()
	mq.conn = nil
	mq.tablesLocked = false
	if err != nil {
		return fmt.Errorf("UNLOCK TABLES failed: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to release pinned connection: %w", closeErr)
	}

	return nil
}

// Close closes the MySQL connection.
func (mq *MySQLQuiescer) Close() error {
	// Ensure tables are unlocked
	if mq.tablesLocked {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mq.Resume(ctx)
	}
	return mq.db.Close()
}

// MongoDBQuiescer handles MongoDB quiesce operations.
type MongoDBQuiescer struct {
	client      *mongo.Client
	database    string
	fsyncLocked bool
}

// NewMongoDBQuiescer creates a new MongoDB quiescer.
func NewMongoDBQuiescer(connString, database string) (*MongoDBQuiescer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connString))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoDBQuiescer{
		client:   client,
		database: database,
	}, nil
}

// Quiesce locks MongoDB with fsync.
func (mq *MongoDBQuiescer) Quiesce(ctx context.Context) (*QuiesceOperation, error) {
	op := &QuiesceOperation{
		DatabaseType: "mongodb",
		DatabaseName: mq.database,
		State:        StateQuiescing,
		StartTime:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Execute db.fsyncLock()
	// This forces a flush of all pending writes and locks the database
	adminDB := mq.client.Database("admin")

	var result struct {
		OK        int    `bson:"ok"`
		Info      string `bson:"info"`
		LockCount int    `bson:"lockCount"`
	}

	command := map[string]interface{}{"fsync": 1, "lock": true}
	err := adminDB.RunCommand(ctx, command).Decode(&result)
	if err != nil {
		op.State = StateFailed
		op.Error = fmt.Errorf("db.fsyncLock() failed: %w", err)
		return op, op.Error
	}

	if result.OK != 1 {
		op.State = StateFailed
		op.Error = fmt.Errorf("db.fsyncLock() returned OK=%d", result.OK)
		return op, op.Error
	}

	mq.fsyncLocked = true
	op.QuiesceTime = time.Now()
	op.State = StateQuiesced
	op.Metadata["lock_duration_ms"] = op.QuiesceTime.Sub(op.StartTime).Milliseconds()
	op.Metadata["lock_count"] = result.LockCount
	op.Metadata["info"] = result.Info

	// Get database stats
	db := mq.client.Database(mq.database)
	var stats struct {
		DataSize    int64 `bson:"dataSize"`
		StorageSize int64 `bson:"storageSize"`
		IndexSize   int64 `bson:"indexSize"`
		Collections int   `bson:"collections"`
		Objects     int64 `bson:"objects"`
	}

	if err := db.RunCommand(ctx, map[string]interface{}{"dbStats": 1}).Decode(&stats); err == nil {
		op.Metadata["data_size_bytes"] = stats.DataSize
		op.Metadata["storage_size_bytes"] = stats.StorageSize
		op.Metadata["index_size_bytes"] = stats.IndexSize
		op.Metadata["collection_count"] = stats.Collections
		op.Metadata["object_count"] = stats.Objects
	}

	// Get replication info if available
	var replStatus struct {
		OK  int    `bson:"ok"`
		Set string `bson:"set"`
	}
	if err := adminDB.RunCommand(ctx, map[string]interface{}{"replSetGetStatus": 1}).Decode(&replStatus); err == nil {
		op.Metadata["replica_set"] = replStatus.Set
	}

	return op, nil
}

// Resume unlocks MongoDB.
func (mq *MongoDBQuiescer) Resume(ctx context.Context) error {
	if !mq.fsyncLocked {
		return nil
	}

	// Execute db.fsyncUnlock()
	adminDB := mq.client.Database("admin")

	var result struct {
		OK   int    `bson:"ok"`
		Info string `bson:"info"`
	}

	command := map[string]interface{}{"fsyncUnlock": 1}
	err := adminDB.RunCommand(ctx, command).Decode(&result)
	if err != nil {
		return fmt.Errorf("db.fsyncUnlock() failed: %w", err)
	}

	if result.OK != 1 {
		return fmt.Errorf("db.fsyncUnlock() returned OK=%d", result.OK)
	}

	mq.fsyncLocked = false
	return nil
}

// Close closes the MongoDB connection.
func (mq *MongoDBQuiescer) Close() error {
	// Ensure database is unlocked
	if mq.fsyncLocked {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mq.Resume(ctx)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return mq.client.Disconnect(ctx)
}

// Quiescer interface for database-agnostic quiesce operations.
type Quiescer interface {
	Quiesce(ctx context.Context) (*QuiesceOperation, error)
	Resume(ctx context.Context) error
	Close() error
}

// QuiesceDatabase performs a quiesce operation on a database.
func (qm *QuiesceManager) QuiesceDatabase(ctx context.Context, quiescer Quiescer, dbID string) (*QuiesceOperation, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Check if database is already quiesced
	if op, exists := qm.operations[dbID]; exists && op.State == StateQuiesced {
		return nil, fmt.Errorf("database %s is already quiesced", dbID)
	}

	// Perform quiesce
	op, err := quiescer.Quiesce(ctx)
	if err != nil {
		return op, err
	}

	// Store operation
	qm.operations[dbID] = op
	return op, nil
}

// ResumeDatabase resumes a quiesced database.
func (qm *QuiesceManager) ResumeDatabase(ctx context.Context, quiescer Quiescer, dbID string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	op, exists := qm.operations[dbID]
	if !exists {
		return fmt.Errorf("no quiesce operation found for database %s", dbID)
	}

	if op.State != StateQuiesced {
		return fmt.Errorf("database %s is not in quiesced state (current: %s)", dbID, op.State)
	}

	op.State = StateResuming
	err := quiescer.Resume(ctx)
	if err != nil {
		op.State = StateFailed
		op.Error = err
		return err
	}

	op.ResumeTime = time.Now()
	op.State = StateNormal
	op.Metadata["total_duration_ms"] = op.ResumeTime.Sub(op.StartTime).Milliseconds()

	return nil
}

// GetOperation returns the quiesce operation for a database.
func (qm *QuiesceManager) GetOperation(dbID string) (*QuiesceOperation, bool) {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	op, exists := qm.operations[dbID]
	return op, exists
}

// GetAllOperations returns all quiesce operations.
func (qm *QuiesceManager) GetAllOperations() map[string]*QuiesceOperation {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	operations := make(map[string]*QuiesceOperation)
	for k, v := range qm.operations {
		operations[k] = v
	}
	return operations
}

// ClearOperation removes a quiesce operation.
func (qm *QuiesceManager) ClearOperation(dbID string) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	delete(qm.operations, dbID)
}
