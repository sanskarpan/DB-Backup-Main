package storageregistry

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

// storeFileName is the single JSON file that holds all registered providers.
const storeFileName = "storage_providers.json"

// storedRecord is the on-disk representation of a StorageProvider. Unlike the
// API-facing StorageProvider type, it DOES persist secret configuration (so it
// survives a restart), but those secrets still never reach a client, because
// handlers only ever serialize StorageProvider values (whose Config holds only
// non-secret keys).
type storedRecord struct {
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Config    map[string]interface{} `json:"config"`
	Secrets   map[string]string      `json:"secrets,omitempty"`
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Enabled   bool                   `json:"enabled"`
	Encrypted bool                   `json:"encrypted"`
}

// Store persists registered storage providers to a JSON file on disk. It is
// safe for concurrent use. Records are loaded on construction so they survive
// restarts.
//
// Secret config values are encrypted at rest with AES-256-GCM when an
// encryption key is supplied to NewStore. If no key is supplied secrets are
// stored in plaintext on disk — but in either case they are NEVER returned in
// an API response, because the StorageProvider.Config map only ever contains
// non-secret keys.
type Store struct {
	records   map[string]*storedRecord
	encryptor *encryption.AESEncryptor
	dir       string
	mu        sync.RWMutex
}

// NewStore creates a Store rooted at dir, creating the directory if needed and
// loading any previously persisted records. If encryptionKey is non-empty,
// secret config values are encrypted at rest.
func NewStore(dir, encryptionKey string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("storageregistry: store directory is required")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("storageregistry: create store dir: %w", err)
	}

	s := &Store{
		dir:     dir,
		records: make(map[string]*storedRecord),
	}

	if encryptionKey != "" {
		enc, err := encryption.NewAESEncryptor([]byte(encryptionKey))
		if err != nil {
			return nil, fmt.Errorf("storageregistry: init encryptor: %w", err)
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
		return fmt.Errorf("storageregistry: read store: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	var recs []*storedRecord
	if err := json.Unmarshal(data, &recs); err != nil {
		return fmt.Errorf("storageregistry: parse store: %w", err)
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
		return fmt.Errorf("storageregistry: marshal store: %w", err)
	}

	tmp, err := os.CreateTemp(s.dir, storeFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("storageregistry: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("storageregistry: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("storageregistry: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storageregistry: close temp: %w", err)
	}
	if err := os.Rename(tmpName, s.path()); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("storageregistry: rename store: %w", err)
	}
	return nil
}

// splitConfig separates an incoming config map into its non-secret part (kept
// in the clear) and its secret part (values converted to strings). Empty secret
// values are dropped so they can fall back to any previously stored secret on
// update.
func splitConfig(cfg map[string]interface{}) (public map[string]interface{}, secrets map[string]string) {
	public = make(map[string]interface{})
	secrets = make(map[string]string)
	for k, v := range cfg {
		if secretConfigKeys[k] {
			str := stringify(v)
			if str != "" {
				secrets[k] = str
			}
			continue
		}
		public[k] = v
	}
	return public, secrets
}

// stringify renders a config value as a string for secret storage.
func stringify(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

// encodeSecrets encrypts (if a key is configured) and hex-encodes each secret
// value, returning the encoded map and whether encryption was applied.
func (s *Store) encodeSecrets(secrets map[string]string) (encoded map[string]string, encrypted bool, err error) {
	if len(secrets) == 0 {
		return nil, false, nil
	}
	out := make(map[string]string, len(secrets))
	if s.encryptor == nil {
		for k, v := range secrets {
			out[k] = v
		}
		return out, false, nil
	}
	for k, v := range secrets {
		ciphertext, encErr := s.encryptor.Encrypt([]byte(v))
		if encErr != nil {
			return nil, false, fmt.Errorf("storageregistry: encrypt secret %q: %w", k, encErr)
		}
		out[k] = hex.EncodeToString(ciphertext)
	}
	return out, true, nil
}

// decodeSecrets reverses encodeSecrets.
func (s *Store) decodeSecrets(r *storedRecord) (map[string]string, error) {
	if len(r.Secrets) == 0 {
		return map[string]string{}, nil
	}
	if !r.Encrypted {
		out := make(map[string]string, len(r.Secrets))
		for k, v := range r.Secrets {
			out[k] = v
		}
		return out, nil
	}
	if s.encryptor == nil {
		return nil, fmt.Errorf("storageregistry: record has encrypted secrets but no encryption key configured")
	}
	out := make(map[string]string, len(r.Secrets))
	for k, v := range r.Secrets {
		raw, err := hex.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("storageregistry: decode secret %q: %w", k, err)
		}
		plain, err := s.encryptor.Decrypt(raw)
		if err != nil {
			return nil, fmt.Errorf("storageregistry: decrypt secret %q: %w", k, err)
		}
		out[k] = string(plain)
	}
	return out, nil
}

// toProvider converts a stored record into an API-facing StorageProvider. The
// returned Config contains only non-secret keys.
func (s *Store) toProvider(r *storedRecord) *StorageProvider {
	cfg := make(map[string]interface{}, len(r.Config))
	for k, v := range r.Config {
		cfg[k] = v
	}
	return &StorageProvider{
		ID:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		Config:    cfg,
		Enabled:   r.Enabled,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// List returns all registered storage providers (without secrets).
func (s *Store) List() ([]*StorageProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*StorageProvider, 0, len(s.records))
	for _, r := range s.records {
		out = append(out, s.toProvider(r))
	}
	return out, nil
}

// Get returns a single storage provider by ID, or ErrNotFound.
func (s *Store) Get(id string) (*StorageProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s.toProvider(r), nil
}

// Resolve returns the full configuration (non-secret values merged with
// decrypted secrets) for internal use such as a real connection test. The
// result is never serialized into an API response.
func (s *Store) Resolve(id string) (*ResolvedProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	secrets, err := s.decodeSecrets(r)
	if err != nil {
		return nil, err
	}
	cfg := make(map[string]interface{}, len(r.Config)+len(secrets))
	for k, v := range r.Config {
		cfg[k] = v
	}
	for k, v := range secrets {
		cfg[k] = v
	}
	return &ResolvedProvider{ID: r.ID, Type: r.Type, Config: cfg}, nil
}

// Create validates and persists a new storage provider, returning it.
func (s *Store) Create(req *CreateRequest) (*StorageProvider, error) {
	if err := validate(req.Name, req.Type); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := newID()
	if err != nil {
		return nil, err
	}
	public, secrets := splitConfig(req.Config)
	encoded, encrypted, err := s.encodeSecrets(secrets)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	r := &storedRecord{
		ID:        id,
		Name:      req.Name,
		Type:      req.Type,
		Config:    public,
		Secrets:   encoded,
		Encrypted: encrypted,
		Enabled:   req.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.records[id] = r

	if err := s.persist(); err != nil {
		delete(s.records, id)
		return nil, err
	}
	return s.toProvider(r), nil
}

// Update validates and applies changes to an existing provider. Any secret key
// that is absent or empty in the request preserves the existing stored secret.
func (s *Store) Update(id string, req *UpdateRequest) (*StorageProvider, error) {
	if err := validate(req.Name, req.Type); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}

	public, newSecrets := splitConfig(req.Config)

	// Start from the existing (decrypted) secrets, then overlay any newly
	// supplied non-empty secret values.
	currentSecrets, err := s.decodeSecrets(existing)
	if err != nil {
		return nil, err
	}
	for k, v := range newSecrets {
		currentSecrets[k] = v
	}
	encoded, encrypted, err := s.encodeSecrets(currentSecrets)
	if err != nil {
		return nil, err
	}

	// Copy so we can roll back on persist failure.
	updated := *existing
	updated.Name = req.Name
	updated.Type = req.Type
	updated.Config = public
	updated.Secrets = encoded
	updated.Encrypted = encrypted
	updated.Enabled = req.Enabled
	updated.UpdatedAt = time.Now().UTC()

	s.records[id] = &updated
	if err := s.persist(); err != nil {
		s.records[id] = existing
		return nil, err
	}
	return s.toProvider(&updated), nil
}

// Delete removes a storage provider by ID, or returns ErrNotFound.
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

// Count returns the number of registered storage providers.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// newID returns a random hex identifier.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("storageregistry: generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
