package secrets

import (
	"path/filepath"
	"testing"
)

func TestSecretSetGetListRevealDelete(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "data", "secrets", "secrets.yaml"))
	if err := store.Set("password", "SuperSecret123"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	value, ok, err := store.Get("password")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok || value != "SuperSecret123" {
		t.Fatalf("Get() = %q, %t", value, ok)
	}
	keys, err := store.Keys()
	if err != nil {
		t.Fatalf("Keys() error = %v", err)
	}
	if len(keys) != 1 || keys[0] != "password" {
		t.Fatalf("Keys() = %#v", keys)
	}
	if err := store.Delete("password"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, ok, err = store.Get("password")
	if err != nil {
		t.Fatalf("Get after delete error = %v", err)
	}
	if ok {
		t.Fatal("secret still exists after delete")
	}
}
