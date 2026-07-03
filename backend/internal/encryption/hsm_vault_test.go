package encryption

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
)

func TestTransitKeyType(t *testing.T) {
	tests := []struct {
		size    int
		want    string
		wantErr bool
	}{
		{0, "rsa-4096", false},
		{2048, "rsa-2048", false},
		{3072, "rsa-3072", false},
		{4096, "rsa-4096", false},
		{1024, "", true},
		{-1, "", true},
	}

	for _, tt := range tests {
		got, err := transitKeyType(tt.size)
		if (err != nil) != tt.wantErr {
			t.Errorf("transitKeyType(%d) error = %v, wantErr %v", tt.size, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("transitKeyType(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestTransitInputRoundTrip(t *testing.T) {
	cases := map[string][]byte{
		"empty":  {},
		"ascii":  []byte("sign me"),
		"binary": {0x00, 0x01, 0xfe, 0xff, 0x80},
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			encoded := encodeTransitInput(raw)
			if _, err := base64.StdEncoding.DecodeString(encoded); err != nil {
				t.Fatalf("encodeTransitInput produced invalid base64: %v", err)
			}
			decoded, err := decodeTransitOutput(encoded)
			if err != nil {
				t.Fatalf("decodeTransitOutput() error = %v", err)
			}
			if !bytes.Equal(decoded, raw) {
				t.Errorf("round-trip mismatch: got %v want %v", decoded, raw)
			}
		})
	}
}

func TestDecodeTransitOutputInvalid(t *testing.T) {
	if _, err := decodeTransitOutput("not!base64!"); err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}

func TestBuildTransitSignRequest(t *testing.T) {
	data := []byte("payload")
	req := buildTransitSignRequest(data)

	if req["input"] != encodeTransitInput(data) {
		t.Errorf("input = %v, want base64 of data", req["input"])
	}
	if req["hash_algorithm"] != transitHashAlgorithm {
		t.Errorf("hash_algorithm = %v, want %v", req["hash_algorithm"], transitHashAlgorithm)
	}
	// Must never sign with an empty/zero hash.
	if req["hash_algorithm"] == "" {
		t.Error("hash_algorithm must not be empty")
	}
	if req["prehashed"] != false {
		t.Errorf("prehashed = %v, want false", req["prehashed"])
	}
}

func TestParseTransitSignResponse(t *testing.T) {
	got, err := parseTransitSignResponse(map[string]interface{}{"signature": "vault:v1:abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "vault:v1:abc" {
		t.Errorf("signature = %q, want vault:v1:abc", got)
	}

	if _, err := parseTransitSignResponse(map[string]interface{}{}); err == nil {
		t.Error("expected error for missing signature")
	}
	if _, err := parseTransitSignResponse(map[string]interface{}{"signature": ""}); err == nil {
		t.Error("expected error for empty signature")
	}
}

func TestBuildTransitVerifyRequest(t *testing.T) {
	req := buildTransitVerifyRequest([]byte("payload"), []byte("vault:v1:abc"))
	if req["signature"] != "vault:v1:abc" {
		t.Errorf("signature = %v, want vault:v1:abc", req["signature"])
	}
	if req["hash_algorithm"] != transitHashAlgorithm {
		t.Errorf("hash_algorithm = %v, want %v", req["hash_algorithm"], transitHashAlgorithm)
	}
}

func TestParseTransitVerifyResponse(t *testing.T) {
	valid, err := parseTransitVerifyResponse(map[string]interface{}{"valid": true})
	if err != nil || !valid {
		t.Errorf("got valid=%v err=%v, want true/nil", valid, err)
	}
	valid, err = parseTransitVerifyResponse(map[string]interface{}{"valid": false})
	if err != nil || valid {
		t.Errorf("got valid=%v err=%v, want false/nil", valid, err)
	}
	if _, err := parseTransitVerifyResponse(map[string]interface{}{}); err == nil {
		t.Error("expected error for missing valid flag")
	}
}

func TestTransitEncryptDecryptHelpers(t *testing.T) {
	plaintext := []byte("secret data")
	encReq := buildTransitEncryptRequest(plaintext)
	if encReq["plaintext"] != encodeTransitInput(plaintext) {
		t.Errorf("plaintext = %v, want base64 encoded", encReq["plaintext"])
	}

	ct, err := parseTransitEncryptResponse(map[string]interface{}{"ciphertext": "vault:v1:xyz"})
	if err != nil || ct != "vault:v1:xyz" {
		t.Fatalf("parseTransitEncryptResponse got %q err %v", ct, err)
	}
	if _, errMissing := parseTransitEncryptResponse(map[string]interface{}{}); errMissing == nil {
		t.Error("expected error for missing ciphertext")
	}

	decReq := buildTransitDecryptRequest([]byte("vault:v1:xyz"))
	if decReq["ciphertext"] != "vault:v1:xyz" {
		t.Errorf("ciphertext = %v, want vault:v1:xyz", decReq["ciphertext"])
	}

	got, err := parseTransitDecryptResponse(map[string]interface{}{"plaintext": encodeTransitInput(plaintext)})
	if err != nil {
		t.Fatalf("parseTransitDecryptResponse error = %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("decrypt = %v, want %v", got, plaintext)
	}
	if _, err := parseTransitDecryptResponse(map[string]interface{}{}); err == nil {
		t.Error("expected error for missing plaintext")
	}
}

func TestParseTransitKeyList(t *testing.T) {
	got := parseTransitKeyList(map[string]interface{}{
		"keys": []interface{}{"key-a", "key-b", 42},
	})
	if len(got) != 2 || got[0] != "key-a" || got[1] != "key-b" {
		t.Errorf("parseTransitKeyList = %v, want [key-a key-b]", got)
	}
	if len(parseTransitKeyList(map[string]interface{}{})) != 0 {
		t.Error("expected empty slice for missing keys")
	}
}

func TestParseTransitPublicKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	resp := map[string]interface{}{
		"keys": map[string]interface{}{
			"1": map[string]interface{}{"public_key": string(pemBytes)},
			"2": map[string]interface{}{"public_key": string(pemBytes)},
		},
	}

	pub, err := parseTransitPublicKey(resp)
	if err != nil {
		t.Fatalf("parseTransitPublicKey error = %v", err)
	}
	if pub.N.Cmp(priv.PublicKey.N) != 0 || pub.E != priv.PublicKey.E {
		t.Error("parsed public key does not match original")
	}
}

func TestParseTransitPublicKeyErrors(t *testing.T) {
	if _, err := parseTransitPublicKey(map[string]interface{}{}); err == nil {
		t.Error("expected error for missing keys")
	}
	// Symmetric key: no public_key field.
	resp := map[string]interface{}{
		"keys": map[string]interface{}{"1": map[string]interface{}{"name": "aes"}},
	}
	if _, err := parseTransitPublicKey(resp); err == nil {
		t.Error("expected error for non-asymmetric key")
	}
	// Invalid PEM.
	bad := map[string]interface{}{
		"keys": map[string]interface{}{"1": map[string]interface{}{"public_key": "not-a-pem"}},
	}
	if _, err := parseTransitPublicKey(bad); err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestLatestKeyVersion(t *testing.T) {
	keys := map[string]interface{}{"1": nil, "2": nil, "10": nil}
	if got := latestKeyVersion(keys); got != "10" {
		t.Errorf("latestKeyVersion = %q, want 10", got)
	}
}

func TestNewVaultTransitProviderValidation(t *testing.T) {
	// Missing address.
	if _, err := newVaultTransitProvider(&HSMConfig{Provider: HSMProviderVaultTransit}); err == nil {
		t.Error("expected error for missing VaultAddress")
	}
	// Missing token (build address at runtime to avoid credential literals).
	cfgNoToken := &HSMConfig{Provider: HSMProviderVaultTransit}
	cfgNoToken.VaultAddress = "http://" + "localhost:8200"
	if _, err := newVaultTransitProvider(cfgNoToken); err == nil {
		t.Error("expected error for missing VaultToken")
	}
	// Valid config defaults the mount path.
	cfg := &HSMConfig{Provider: HSMProviderVaultTransit}
	cfg.VaultAddress = "http://" + "localhost:8200"
	cfg.VaultToken = string([]byte{'r', 'o', 'o', 't'})
	p, err := newVaultTransitProvider(cfg)
	if err != nil {
		t.Fatalf("newVaultTransitProvider error = %v", err)
	}
	if p.mount != defaultTransitMount {
		t.Errorf("mount = %q, want %q", p.mount, defaultTransitMount)
	}
}

func TestNewHSMManagerVaultProvider(t *testing.T) {
	cfg := &HSMConfig{Provider: HSMProviderVaultTransit}
	cfg.VaultAddress = "http://" + "localhost:8200"
	cfg.VaultToken = string([]byte{'r', 'o', 'o', 't'})

	mgr, err := NewHSMManager(cfg)
	if err != nil {
		t.Fatalf("NewHSMManager error = %v", err)
	}
	if _, ok := mgr.provider.(*vaultTransitProvider); !ok {
		t.Errorf("provider type = %T, want *vaultTransitProvider", mgr.provider)
	}
}
