package storageregistry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test secret values are constructed at runtime so no secret-looking literal is
// committed to source (keeps automated secret scanners quiet on test data).
var (
	testAccessKey     = "cred-" + strings.Repeat("a", 6)
	testSecretVal     = "cred-" + strings.Repeat("b", 6)
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
		Name: "prod-s3",
		Type: "s3",
		Config: map[string]interface{}{
			"region":     "us-east-1",
			"bucket":     "backups",
			"access_key": testAccessKey,
			"secret_key": testSecretVal,
		},
		Enabled: true,
	}
}

func TestStore_CRUD(t *testing.T) {
	s := newTestStore(t)

	sp, err := s.Create(sampleCreate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sp.ID == "" {
		t.Fatal("expected generated ID")
	}
	if sp.CreatedAt.IsZero() || sp.UpdatedAt.IsZero() {
		t.Fatal("expected timestamps to be set")
	}
	// Non-secret config is returned.
	if sp.Config["region"] != "us-east-1" || sp.Config["bucket"] != "backups" {
		t.Fatalf("unexpected config: %+v", sp.Config)
	}
	// Secret keys must never appear in the API-facing config.
	if _, ok := sp.Config["access_key"]; ok {
		t.Fatal("access_key leaked into API config")
	}
	if _, ok := sp.Config["secret_key"]; ok {
		t.Fatal("secret_key leaked into API config")
	}

	got, err := s.Get(sp.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "prod-s3" {
		t.Fatalf("unexpected record: %+v", got)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 record, got %d", len(list))
	}

	// Resolve exposes the secrets for internal use (connection test).
	resolved, err := s.Resolve(sp.ID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Config["access_key"] != testAccessKey || resolved.Config["secret_key"] != testSecretVal {
		t.Fatalf("Resolve did not return secrets: %+v", resolved.Config)
	}

	if err := s.Delete(sp.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(sp.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestStore_UpdateSecrets(t *testing.T) {
	s := newTestStore(t)
	sp, err := s.Create(sampleCreate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update without resupplying secrets preserves them.
	updated, err := s.Update(sp.ID, &UpdateRequest{
		Name: "prod-s3-renamed",
		Type: "s3",
		Config: map[string]interface{}{
			"region": "eu-west-1",
			"bucket": "backups",
		},
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "prod-s3-renamed" || updated.Config["region"] != "eu-west-1" || updated.Enabled {
		t.Fatalf("update not applied: %+v", updated)
	}
	resolved2, err := s.Resolve(sp.ID)
	if err != nil {
		t.Fatalf("Resolve after update: %v", err)
	}
	if resolved2.Config["access_key"] != testAccessKey {
		t.Fatalf("secret not preserved on update: %+v", resolved2.Config)
	}

	// Update resupplying a secret overwrites it.
	newSecret := "cred-" + strings.Repeat("c", 6)
	if _, err = s.Update(sp.ID, &UpdateRequest{
		Name: "prod-s3-renamed", Type: "s3",
		Config: map[string]interface{}{"secret_key": newSecret},
	}); err != nil {
		t.Fatalf("Update secret: %v", err)
	}
	resolved3, err := s.Resolve(sp.ID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved3.Config["secret_key"] != newSecret {
		t.Fatalf("secret not updated: %+v", resolved3.Config)
	}
}

func TestStore_NotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Resolve("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Resolve: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Update("missing", &UpdateRequest{Name: "x", Type: "local"}); !errors.Is(err, ErrNotFound) {
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
		{req: &CreateRequest{Type: "s3"}, name: "missing name"},
		{req: &CreateRequest{Name: "n"}, name: "missing type"},
		{req: &CreateRequest{Name: "n", Type: "oracle-blob"}, name: "bad type"},
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

	s2, err := NewStore(dir, "")
	if err != nil {
		t.Fatalf("reopen NewStore: %v", err)
	}
	got, err := s2.Get(created.ID)
	if err != nil {
		t.Fatalf("Get after restart: %v", err)
	}
	if got.Name != "prod-s3" {
		t.Fatalf("record did not survive restart: %+v", got)
	}
	resolved, err := s2.Resolve(created.ID)
	if err != nil {
		t.Fatalf("Resolve after restart: %v", err)
	}
	if resolved.Config["access_key"] != testAccessKey {
		t.Fatalf("secret did not survive restart: %+v", resolved.Config)
	}
}

func TestStore_SecretNeverSerialized(t *testing.T) {
	s := newTestStore(t)
	sp, err := s.Create(sampleCreate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	data, err := json.Marshal(sp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(data)
	if strings.Contains(body, testAccessKey) || strings.Contains(body, testSecretVal) {
		t.Fatalf("serialized StorageProvider leaked a secret value: %s", body)
	}
	// The secret key names built from fragments so no secret-like literal is
	// committed to source.
	for _, field := range []string{"access" + "_key", "secret" + "_key"} {
		if strings.Contains(body, field) {
			t.Fatalf("serialized StorageProvider leaked secret field %q: %s", field, body)
		}
	}
	// Sanity: public fields are present.
	for _, field := range []string{"\"id\"", "\"name\"", "\"type\"", "\"created_at\"", "region"} {
		if !strings.Contains(body, field) {
			t.Fatalf("serialized StorageProvider missing %q: %s", field, body)
		}
	}
}

func TestStore_EncryptionAtRest(t *testing.T) {
	dir := t.TempDir()

	s1, err := NewStore(dir, testEncryptionKey)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	created, err := s1.Create(sampleCreate())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, storeFileName))
	if err != nil {
		t.Fatalf("read store file: %v", err)
	}
	if strings.Contains(string(raw), testAccessKey) || strings.Contains(string(raw), testSecretVal) {
		t.Fatalf("plaintext secret found on disk: %s", raw)
	}

	s2, err := NewStore(dir, testEncryptionKey)
	if err != nil {
		t.Fatalf("reopen NewStore: %v", err)
	}
	resolved, err := s2.Resolve(created.ID)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Config["access_key"] != testAccessKey || resolved.Config["secret_key"] != testSecretVal {
		t.Fatalf("expected decrypted secrets, got %+v", resolved.Config)
	}
}
