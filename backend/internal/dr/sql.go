// Package dr provides disaster recovery testing and validation.
//
// This file contains the low-level, real database helpers used by the DR
// executor and validator: DSN construction, connection opening, statement
// splitting for restores, identifier quoting and driver-aware metadata
// queries. Everything here talks to a real database/sql target -- there is
// no simulation.
package dr

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	// Register the SQL drivers that DR targets can use. These are all already
	// direct dependencies of the module.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Driver names as registered with database/sql.
const (
	driverSQLite   = "sqlite3"
	driverPostgres = "postgres"
	driverMySQL    = "mysql"
)

// normalizeDriver maps user-facing driver aliases to the database/sql driver
// name that is actually registered.
func normalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlite", driverSQLite:
		return driverSQLite
	case driverPostgres, "postgresql", "pgx", "pg":
		return driverPostgres
	case driverMySQL, "mariadb":
		return driverMySQL
	default:
		return strings.ToLower(strings.TrimSpace(driver))
	}
}

// buildDSN constructs a database/sql DSN from a TargetConfig. If an explicit
// DSN is provided it is used verbatim.
func buildDSN(target *TargetConfig) (string, error) {
	if target == nil {
		return "", fmt.Errorf("no test target configured")
	}
	if strings.TrimSpace(target.DSN) != "" {
		return target.DSN, nil
	}

	driver := normalizeDriver(target.Driver)
	switch driver {
	case driverSQLite:
		return "", fmt.Errorf("sqlite target requires an explicit DSN (database file path or :memory:)")
	case driverPostgres:
		sslMode := target.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			target.Host, target.Port, target.User, target.Password, target.Database, sslMode,
		), nil
	case driverMySQL:
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?multiStatements=true&parseTime=true",
			target.User, target.Password, target.Host, target.Port, target.Database,
		), nil
	default:
		return "", fmt.Errorf("unsupported target driver: %q", target.Driver)
	}
}

// openTarget opens (and verifies connectivity to) a real database connection
// for the given target.
func openTarget(ctx context.Context, target *TargetConfig) (*sql.DB, error) {
	if target == nil {
		return nil, fmt.Errorf("no test target configured")
	}

	driver := normalizeDriver(target.Driver)
	dsn, err := buildDSN(target)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s target: %w", driver, err)
	}

	// SQLite in-memory / file databases only retain state on a single
	// underlying connection, so pin the pool to one connection to make the
	// restore visible to subsequent validation queries.
	if driver == driverSQLite {
		db.SetMaxOpenConns(1)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping %s target: %w", driver, err)
	}

	return db, nil
}

// quoteIdent quotes a SQL identifier for the given driver, doubling the
// closing quote character to guard against injection through object names.
func quoteIdent(driver, name string) string {
	switch normalizeDriver(driver) {
	case driverMySQL:
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	default: // postgres, sqlite3 and the SQL standard use double quotes
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}
}

// splitSQLStatements splits a SQL script into individual statements on
// unquoted semicolons. Single-quoted string literals, double-quoted
// identifiers and -- line comments are respected so that semicolons inside
// them do not prematurely terminate a statement. This is sufficient for the
// logical SQL dumps produced/consumed by the platform's backup drivers.
func splitSQLStatements(script string) []string {
	var (
		statements []string
		current    strings.Builder
		state      sqlScanState
	)

	runes := []rune(script)
	for i := 0; i < len(runes); i++ {
		c := runes[i]

		if state.inLineComm {
			current.WriteRune(c)
			if c == '\n' {
				state.inLineComm = false
			}
			continue
		}

		state.update(runes, i)

		if c == ';' && !state.inSingle && !state.inDouble {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteRune(c)
	}

	if stmt := strings.TrimSpace(current.String()); stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// sqlScanState tracks the quoting/comment context while scanning a SQL script
// so that semicolons inside string literals, quoted identifiers or line
// comments are not treated as statement terminators.
type sqlScanState struct {
	inSingle   bool
	inDouble   bool
	inLineComm bool
}

// update advances the scan state for the rune at runes[i], toggling
// single-quote, double-quote or line-comment context as appropriate.
func (s *sqlScanState) update(runes []rune, i int) {
	c := runes[i]
	switch {
	case c == '\'' && !s.inDouble:
		s.inSingle = !s.inSingle
	case c == '"' && !s.inSingle:
		s.inDouble = !s.inDouble
	case c == '-' && !s.inSingle && !s.inDouble && i+1 < len(runes) && runes[i+1] == '-':
		s.inLineComm = true
	}
}

// listTables returns the base tables that exist in the connected database,
// using a driver-appropriate metadata query.
func listTables(ctx context.Context, db *sql.DB, driver, database string) ([]string, error) {
	var (
		query string
		args  []interface{}
	)

	switch normalizeDriver(driver) {
	case driverSQLite:
		query = `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	case driverPostgres:
		query = `SELECT table_name FROM information_schema.tables
			WHERE table_type = 'BASE TABLE'
			AND table_schema NOT IN ('pg_catalog', 'information_schema')
			ORDER BY table_name`
	case driverMySQL:
		query = `SELECT table_name FROM information_schema.tables
			WHERE table_type = 'BASE TABLE' AND table_schema = ?
			ORDER BY table_name`
		args = append(args, database)
	default:
		return nil, fmt.Errorf("unsupported target driver: %q", driver)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tables: %w", err)
	}

	return tables, nil
}

// tableRowCount runs a real COUNT(*) against the given table.
func tableRowCount(ctx context.Context, db *sql.DB, driver, table string) (int64, error) {
	query := "SELECT COUNT(*) FROM " + quoteIdent(driver, table)
	var count int64
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count rows in %s: %w", table, err)
	}
	return count, nil
}

// applySQLScript executes every statement in a SQL script inside a single
// transaction, rolling back on the first failure. It returns the number of
// statements applied.
func applySQLScript(ctx context.Context, db *sql.DB, script string) (int, error) {
	statements := splitSQLStatements(script)
	if len(statements) == 0 {
		return 0, fmt.Errorf("backup contains no executable SQL statements")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin restore transaction: %w", err)
	}

	applied := 0
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return applied, fmt.Errorf("apply statement %d: %w (rollback failed: %v)", applied+1, err, rbErr)
			}
			return applied, fmt.Errorf("apply statement %d: %w", applied+1, err)
		}
		applied++
	}

	if err := tx.Commit(); err != nil {
		return applied, fmt.Errorf("commit restore transaction: %w", err)
	}

	return applied, nil
}

// dropAllTables scrubs every base table from the connected database. It is
// used to clean up a test target after a DR drill.
func dropAllTables(ctx context.Context, db *sql.DB, driver, database string) error {
	tables, err := listTables(ctx, db, driver, database)
	if err != nil {
		return err
	}
	if len(tables) == 0 {
		return nil
	}

	var firstErr error

	// MySQL/PostgreSQL may have FK constraints; disable checks where possible.
	if normalizeDriver(driver) == driverMySQL {
		if _, err := db.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err == nil {
			defer func() {
				if _, rerr := db.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 1"); rerr != nil && firstErr == nil {
					firstErr = fmt.Errorf("re-enable foreign key checks: %w", rerr)
				}
			}()
		}
	}

	for _, table := range tables {
		stmt := "DROP TABLE IF EXISTS " + quoteIdent(driver, table)
		if normalizeDriver(driver) == driverPostgres {
			stmt += " CASCADE"
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("drop table %s: %w", table, err)
		}
	}

	return firstErr
}
