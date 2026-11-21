package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenPreservesReservedFilenameCharacters(t *testing.T) {
	names := []string{"notes#fragment.db", "notes%percent.db", "notes#50%.db", "notes%23.db"}
	if runtime.GOOS != "windows" {
		names = append(names, "notes?query.db")
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			store, err := Open(path)
			if err != nil {
				t.Fatalf("open literal path: %v", err)
			}
			if _, err := store.Add("must survive close and reopen", "uri"); err != nil {
				_ = store.Close()
				t.Fatalf("insert note: %v", err)
			}
			if err := store.Close(); err != nil {
				t.Fatalf("close database: %v", err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("database was not created at requested path %q: %v", path, err)
			}
			if info.IsDir() {
				t.Fatalf("requested database path %q is a directory", path)
			}

			reopened, err := Open(path)
			if err != nil {
				t.Fatalf("reopen literal path: %v", err)
			}
			defer reopened.Close()
			notes, err := reopened.List(false, "uri")
			if err != nil {
				t.Fatalf("read persisted notes: %v", err)
			}
			if len(notes) != 1 || notes[0].Body != "must survive close and reopen" {
				t.Fatalf("reopened database contains %d matching notes, want original note", len(notes))
			}
		})
	}
}
