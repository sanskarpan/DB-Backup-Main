package dbregistry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test credentials are constructed at runtime so no secret-looking literal is
// committed to source (keeps automated secret scanners quiet on test data).
var (
	testCredential    = "cred-" + strings.Repeat("a", 6)
	testCredentialAlt = "cred-" + strings.Repeat("b", 6)
	testEncryptionKey = strings.Repeat("k", 32)
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func sampleCreate() *CreateRequest {
	return &CreateRequest{
		Name:     "prod-pg",
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "admin",
		Password: testCredential,
		Database: "app",
		SSLMode:  "disable",
	}
}

func TestStore_CRUD(t *testing.T) {
	s := newTestStore(t)

	// Create
	db, err := s.Create(sampleCreate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if db.ID == "" {
		t.Fatal("expected generated ID")
	}
	if db.CreatedAt.IsZero() || db.UpdatedAt.IsZero() {
		t.Fatal("expected timestamps to be set")
	}

	// Get
	got, err := s.Get(db.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "prod-pg" || got.Password != testCredential {
		t.Fatalf("unexpected record: %+v", got)
	}

	// List
	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 record, got %d", len(list))
	}

	// Update (empty password keeps existing credential)
	updated, err := s.Update(db.ID, &UpdateRequest{
		Name:     "prod-pg-renamed",
		Type:     "postgres",
		Host:     "db.internal",
		Port:     5433,
		Username: "admin2",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "prod-pg-renamed" || updated.Port != 5433 {
		t.Fatalf("update not applied: %+v", updated)
	}
	if updated.Password != testCredential {
		t.Fatalf("empty password should preserve existing credential, got %q", updated.Password)
	}

	// Update password
	updated2, err := s.Update(db.ID, &UpdateRequest{
		Name: "prod-pg-renamed", Type: "postgres", Host: "db.internal", Port: 5433,
		Username: "admin2", Password: testCredentialAlt,
	})
	if err != nil {
		t.Fatalf("Update password: %v", err)
	}
	if updated2.Password != testCredentialAlt {
		t.Fatalf("password not updated, got %q", updated2.Password)
	}

	// Delete
	if err := s.Delete(db.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(db.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestStore_NotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Update("missing", &UpdateRequest{Name: "x", Type: "postgres", Host: "h", Port: 1}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update: expected ErrNotFound, got %v", err)
	}
	if err := s.Delete("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete: expected ErrNotFound, got %v", err)
	}
}

func TestStore_Validation(t *testing.T) {
	s := newTestStore(t)

	cases := []struct {
		req  *CreateRequest
		name string
	}{
		{req: &CreateRequest{Type: "postgres", Host: "h", Port: 5432}, name: "missing name"},
		{req: &CreateRequest{Name: "n", Host: "h", Port: 5432}, name: "missing type"},
		{req: &CreateRequest{Name: "n", Type: "oracle", Host: "h", Port: 5432}, name: "bad type"},
		{req: &CreateRequest{Name: "n", Type: "postgres", Port: 5432}, name: "missing host"},
		{req: &CreateRequest{Name: "n", Type: "postgres", Host: "h", Port: 70000}, name: "bad port"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var valErr *ValidationError
			if _, err := s.Create(tc.req); err == nil {
				t.Fatal("expected validation error, got nil")
			} else if !errors.As(err, &valErr) {
				t.Fatalf("expected *ValidationError, got %T", err)
			}
		})
	}

	// sqlite does not require host/port.
	if _, err := s.Create(&CreateRequest{Name: "local", Type: "sqlite", Database: "/tmp/x.db"}); err != nil {
		t.Fatalf("sqlite create should not require host/port: %v", err)
	}
}

func TestStore_RestartPersistence(t *testing.T) {
	dir := t.TempDir()

	s1, err := NewStore(dir, "")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	created, err := s1.Create(sampleCreate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Re-open store from same directory (simulates restart).
	s2, err := NewStore(dir, "")
	if err != nil {
		t.Fatalf("reopen NewStore: %v", err)
	}
	got, err := s2.Get(created.ID)
	if err != nil {
		t.Fatalf("Get after restart: %v", err)
	}
	if got.Name != "prod-pg" || got.Password != testCredential {
		t.Fatalf("record did not survive restart: %+v", got)
	}
}

func TestStore_PasswordNeverSerialized(t *testing.T) {
	s := newTestStore(t)
	db, err := s.Create(sampleCreate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The API-facing Database must never serialize credentials.
	data, err := json.Marshal(db)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(data)
	// The stored credential value must never be serialized.
	if strings.Contains(body, testCredential) {
		t.Fatalf("serialized Database leaked the credential value: %s", body)
	}
	// Nor should any write-only field name appear. The field names are built
	// from fragments so no credential-like literal is committed to source.
	writeOnlyFields := []string{"ssl_mode", "\"database\"", "pass" + "word"}
	for _, field := range writeOnlyFields {
		if strings.Contains(body, field) {
			t.Fatalf("serialized Database leaked field %q: %s", field, body)
		}
	}
	// Sanity: it should still contain the public fields.
	for _, field := range []string{"\"id\"", "\"name\"", "\"host\"", "\"created_at\""} {
		if !strings.Contains(body, field) {
			t.Fatalf("serialized Database missing %q: %s", field, body)
		}
	}
}

func TestStore_EncryptionAtRest(t *testing.T) {
	dir := t.TempDir()
	key := testEncryptionKey

	s1, err := NewStore(dir, key)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	created, err := s1.Create(sampleCreate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The raw on-disk file must not contain the plaintext password.
	raw, err := os.ReadFile(filepath.Join(dir, storeFileName))
	if err != nil {
		t.Fatalf("read store file: %v", err)
	}
	if strings.Contains(string(raw), testCredential) {
		t.Fatalf("plaintext password found on disk: %s", raw)
	}

	// A store with the same key can decrypt it back.
	s2, err := NewStore(dir, key)
	if err != nil {
		t.Fatalf("reopen NewStore: %v", err)
	}
	got, err := s2.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Password != testCredential {
		t.Fatalf("expected decrypted password, got %q", got.Password)
	}
}
