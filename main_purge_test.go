package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPurgeRejectsUnexpectedArgumentsWithoutDeletingNotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.db")
	t.Setenv("TICKPOST_DB", path)
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	note, err := store.Add("retain this closed note", "ops")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetDone(note.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"purge", "unexpected"}); err == nil {
		t.Fatal("purge with unexpected argument must fail")
	}
	verify, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer verify.Close()
	notes, err := verify.List(true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].ID != note.ID || !notes[0].Done {
		t.Fatalf("closed note was altered after rejected purge: %+v", notes)
	}
}

func TestRejectedPurgeDoesNotCreateDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "should-not-exist.db")
	t.Setenv("TICKPOST_DB", path)
	if err := run([]string{"purge", "unexpected"}); err == nil {
		t.Fatal("expected a purge argument error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("rejected purge created a database or returned wrong stat error: %v", err)
	}
}

func TestPurgeNoArgumentsDeletesOnlyClosedNotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.db")
	t.Setenv("TICKPOST_DB", path)
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	closed, err := store.Add("closed", "ops")
	if err != nil {
		t.Fatal(err)
	}
	open, err := store.Add("open", "ops")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetDone(closed.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"purge"}); err != nil {
		t.Fatal(err)
	}
	verify, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer verify.Close()
	notes, err := verify.List(true, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].ID != open.ID || notes[0].Done {
		t.Fatalf("purge should retain only the open note: %+v", notes)
	}
}
