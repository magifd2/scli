package keychain_test

import (
	"testing"

	"github.com/magifd2/scli/internal/keychain"
)

func TestMockStore(t *testing.T) {
	store := keychain.NewMockStore()

	// Get on empty store returns error
	if _, err := store.Get("ws1"); err == nil {
		t.Error("expected error on missing key, got nil")
	}

	// Set then Get returns the stored token
	if err := store.Set("ws1", "xoxp-token-1"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	token, err := store.Get("ws1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if token != "xoxp-token-1" {
		t.Errorf("got %q, want %q", token, "xoxp-token-1")
	}

	// Delete removes the entry
	if err := store.Delete("ws1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get("ws1"); err == nil {
		t.Error("expected error after Delete, got nil")
	}

	// Delete on non-existent key is a no-op
	if err := store.Delete("nonexistent"); err != nil {
		t.Errorf("Delete non-existent: %v", err)
	}
}

func TestMockStoreMultipleWorkspaces(t *testing.T) {
	store := keychain.NewMockStore()

	_ = store.Set("ws1", "token-1")
	_ = store.Set("ws2", "token-2")

	t1, _ := store.Get("ws1")
	t2, _ := store.Get("ws2")

	if t1 != "token-1" {
		t.Errorf("ws1: got %q, want %q", t1, "token-1")
	}
	if t2 != "token-2" {
		t.Errorf("ws2: got %q, want %q", t2, "token-2")
	}
}
