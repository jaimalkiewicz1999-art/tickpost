package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnknownCommandDoesNotCreateDatabaseOrParentDirectory(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "absent-directory")
	path := filepath.Join(parent, "notes.db")
	t.Setenv("TICKPOST_DB", path)

	if err := run([]string{"typo-do-not-create-db"}); err == nil {
		t.Fatal("unknown command must fail")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unknown command created a database or returned unexpected stat error: %v", err)
	}
	if _, err := os.Stat(parent); !os.IsNotExist(err) {
		t.Fatalf("unknown command created a data directory or returned unexpected stat error: %v", err)
	}
}
