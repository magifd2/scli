package slack_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nlink-jp/scli/internal/slack"
)

func TestSaveAndLoadCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	type payload struct {
		Name string
		Age  int
	}
	original := payload{Name: "Alice", Age: 30}

	slack.SaveCacheForTest(path, original)

	got, ok := slack.LoadCacheForTest[payload](path, time.Hour)
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if got.Name != original.Name || got.Age != original.Age {
		t.Errorf("got %+v, want %+v", got, original)
	}
}

func TestLoadCache_Miss_NoFile(t *testing.T) {
	dir := t.TempDir()
	_, ok := slack.LoadCacheForTest[[]string](filepath.Join(dir, "missing.json"), time.Hour)
	if ok {
		t.Error("expected cache miss for non-existent file, got hit")
	}
}

func TestLoadCache_Miss_Expired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "expired.json")

	slack.SaveCacheForTest(path, "value")

	// TTL of zero means anything is expired.
	_, ok := slack.LoadCacheForTest[string](path, 0)
	if ok {
		t.Error("expected cache miss for expired entry, got hit")
	}
}
