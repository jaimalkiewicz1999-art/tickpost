package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMalformedKnownCommandsDoNotCreateStorage(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"missing-add-text", []string{"add"}, "nothing to add"},
		{"missing-add-tag", []string{"add", "-tag"}, "-tag needs a value"},
		{"invalid-done-id", []string{"done", "invalid-id"}, "not a note id"},
		{"missing-rm-id", []string{"rm"}, "expected exactly one note id"},
		{"invalid-remove-id", []string{"remove", "0"}, "not a note id"},
		{"missing-open-id", []string{"open"}, "expected exactly one note id"},
		{"invalid-undone-id", []string{"undone", "-2"}, "not a note id"},
		{"list-unknown-flag", []string{"ls", "-not-a-flag"}, "flag provided but not defined"},
		{"list-missing-tag", []string{"ls", "-tag"}, "flag needs an argument"},
		{"list-unexpected-positional", []string{"ls", "unexpected"}, "ls takes no positional arguments"},
		{"list-alias-positional", []string{"list", "unexpected"}, "ls takes no positional arguments"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parent := filepath.Join(t.TempDir(), "missing-parent")
			path := filepath.Join(parent, "notes.db")
			t.Setenv("TICKPOST_DB", path)

			err := run(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("run(%q) error = %v, want %q", tc.args, err, tc.want)
			}
			for _, absent := range []string{parent, path} {
				if _, err := os.Stat(absent); !os.IsNotExist(err) {
					t.Fatalf("rejected command created %s or stat failed: %v", absent, err)
				}
			}
		})
	}
}
