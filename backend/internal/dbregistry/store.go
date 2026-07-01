package dbregistry

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/internal/encryption"
)

// storeFileName is the single JSON file that holds all registered databases.
const storeFileName = "databases.json"

// storedRecord is the on-disk representation of a Database. Unlike the
// API-facing Database type, it DOES persist credentials (so they survive a
// restart) but they still never reach a client, because handlers only ever
// serialize Database values.
type storedRecord struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Host      string            `json:"host"`
	Port      int               `json:"port"`
	Username  string            `json:"username"`
	Password  string            `json:"password"`  // encrypted when Encrypted is true
	Encrypted bool              `json:"encrypted"` // whether Password is AES-encrypted (hex)
	Database  string            `json:"database,omitempty"`
	SSLMode   string            `json:"ssl_mode,omitempty"`
	Options   map[string]string `json:"options,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Store persists registered databases to a JSON file on disk. It is safe for
// concurrent use. Records are loaded on construction so they survive restarts.
//
// Passwords are encrypted at rest with AES-256-GCM when an encryption key is
// supplied to NewStore (main.go passes the JWT secret, which is always present
// and at least 32 chars). If no key is supplied the password is stored in
// plaintext on disk — but in either case it is NEVER returned in an API
// response, because the Database type tags credential fields json:"-".
type Store struct {
	mu        sync.RWMutex
	dir       string
	records   map[string]*storedRecord
	encryptor *encryption.AESEncryptor
}

// NewStore creates a Store rooted at dir, creating the directory if needed and
// loading any previously persisted records. If encryptionKey is non-empty,
// passwords are encrypted at rest.
func NewStore(dir string, encryptionKey string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("dbregistry: store directory is required")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("dbregistry: create store dir: %w", err)
	}

	s := &Store{
		dir:     dir,
		records: make(map[string]*storedRecord),
	}

	if encryptionKey != "" {
		enc, err := encryption.NewAESEncryptor([]byte(encryptionKey))
		if err != nil {
			return nil, fmt.Errorf("dbregistry: init encryptor: %w", err)
		}
		s.encryptor = enc
	}

	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// path returns the full path to the store file.
func (s *Store) path() string {
	return filepath.Join(s.dir, storeFileName)
}

// load reads all records from disk. A missing file is not an error.
func (s *Store) load() error {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("dbregistry: read store: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	var recs []*storedRecord
	if err := json.Unmarshal(data, &recs); err != nil {
		return fmt.Errorf("dbregistry: parse store: %w", err)
	}
	for _, r := range recs {
		s.records[r.ID] = r
	}
	return nil
}

// persist writes all records to disk atomically (temp file + rename).
// Callers must hold s.mu.
func (s *Store) persist() error {
	recs := make([]*storedRecord, 0, len(s.records))
	for _, r := range s.records {
		recs = append(recs, r)
	}

	data, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return fmt.Errorf("dbregistry: marshal store: %w", err)
	}

	tmp, err := os.CreateTemp(s.dir, storeFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("dbregistry: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("dbregistry: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("dbregistry: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("dbregistry: close temp: %w", err)
	}
	if err := os.Rename(tmpName, s.path()); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("dbregistry: rename store: %w", err)
	}
	return nil
}

// encodePassword returns the value to store for a plaintext password.
func (s *Store) encodePassword(plain string) (value string, encrypted bool, err error) {
	if plain == "" {
		return "", false, nil
	}
	if s.encryptor == nil {
		return plain, false, nil
	}
	ciphertext, err := s.encryptor.Encrypt([]byte(plain))
	if err != nil {
		return "", false, fmt.Errorf("dbregistry: encrypt password: %w", err)
	}
	return hex.EncodeToString(ciphertext), true, nil
}

// decodePassword reverses encodePassword.
func (s *Store) decodePassword(r *storedRecord) (string, error) {
	if r.Password == "" || !r.Encrypted {
		return r.Password, nil
	}
	if s.encryptor == nil {
		return "", fmt.Errorf("dbregistry: record is encrypted but no encryption key configured")
	}
	raw, err := hex.DecodeString(r.Password)
	if err != nil {
		return "", fmt.Errorf("dbregistry: decode password: %w", err)
	}
	plain, err := s.encryptor.Decrypt(raw)
	if err != nil {
		return "", fmt.Errorf("dbregistry: decrypt password: %w", err)
	}
	return string(plain), nil
}

// toDatabase converts a stored record into an API-facing Database, decrypting
// the password into the write-only Password field for internal use (e.g. the
// connection test). The password is still never serialized to a client.
func (s *Store) toDatabase(r *storedRecord) (*Database, error) {
	pw, err := s.decodePassword(r)
	if err != nil {
		return nil, err
	}
	return &Database{
		ID:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		Host:      r.Host,
		Port:      r.Port,
		Username:  r.Username,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		Password:  pw,
		Database:  r.Database,
		SSLMode:   r.SSLMode,
		Options:   r.Options,
	}, nil
}

// List returns all registered databases (without credentials in JSON).
func (s *Store) List() ([]*Database, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Database, 0, len(s.records))
	for _, r := range s.records {
		db, err := s.toDatabase(r)
		if err != nil {
			return nil, err
		}
		out = append(out, db)
	}
	return out, nil
}

// Get returns a single database by ID, or ErrNotFound.
func (s *Store) Get(id string) (*Database, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s.toDatabase(r)
}

// Create validates and persists a new database, returning it.
func (s *Store) Create(req *CreateRequest) (*Database, error) {
	if err := validate(req.Name, req.Type, req.Host, req.Port); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := newID()
	if err != nil {
		return nil, err
	}
	pwValue, encrypted, err := s.encodePassword(req.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	r := &storedRecord{
		ID:        id,
		Name:      req.Name,
		Type:      req.Type,
		Host:      req.Host,
		Port:      req.Port,
		Username:  req.Username,
		Password:  pwValue,
		Encrypted: encrypted,
		Database:  req.Database,
		SSLMode:   req.SSLMode,
		Options:   req.Options,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.records[id] = r

	if err := s.persist(); err != nil {
		delete(s.records, id)
		return nil, err
	}
	return s.toDatabase(r)
}

// Update validates and applies changes to an existing database. An empty
// password leaves the existing credential unchanged.
func (s *Store) Update(id string, req *UpdateRequest) (*Database, error) {
	if err := validate(req.Name, req.Type, req.Host, req.Port); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}

	// Copy so we can roll back on persist failure.
	updated := *existing
	updated.Name = req.Name
	updated.Type = req.Type
	updated.Host = req.Host
	updated.Port = req.Port
	updated.Username = req.Username
	updated.Database = req.Database
	updated.SSLMode = req.SSLMode
	updated.Options = req.Options
	updated.UpdatedAt = time.Now().UTC()

	if req.Password != "" {
		pwValue, encrypted, err := s.encodePassword(req.Password)
		if err != nil {
			return nil, err
		}
		updated.Password = pwValue
		updated.Encrypted = encrypted
	}

	s.records[id] = &updated
	if err := s.persist(); err != nil {
		s.records[id] = existing
		return nil, err
	}
	return s.toDatabase(&updated)
}

// Delete removes a database by ID, or returns ErrNotFound.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.records[id]
	if !ok {
		return ErrNotFound
	}
	delete(s.records, id)
	if err := s.persist(); err != nil {
		s.records[id] = existing
		return err
	}
	return nil
}

// newID returns a random hex identifier.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("dbregistry: generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
