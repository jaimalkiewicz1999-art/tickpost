package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a note id does not exist in the database.
var ErrNotFound = errors.New("note not found")

// Note is a single sticky note. ClosedAt is the zero time while the note is open.
type Note struct {
	ID       int64
	Body     string
	Tag      string
	Done     bool
	AddedAt  time.Time
	ClosedAt time.Time
}

const schema = `
CREATE TABLE IF NOT EXISTS notes (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	body      TEXT    NOT NULL,
	tag       TEXT    NOT NULL DEFAULT '',
	done      INTEGER NOT NULL DEFAULT 0,
	added_at  TEXT    NOT NULL,
	closed_at TEXT
);
CREATE INDEX IF NOT EXISTS notes_open_idx ON notes (done, added_at);
CREATE INDEX IF NOT EXISTS notes_tag_idx  ON notes (tag) WHERE tag <> '';
`

// Store wraps the SQLite database holding the notes.
type Store struct {
	db *sql.DB
}

// DefaultPath honours TICKPOST_DB, then XDG_DATA_HOME, then ~/.local/share.
func DefaultPath() (string, error) {
	if p := os.Getenv("TICKPOST_DB"); p != "" {
		return p, nil
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate home directory: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "tickpost", "tickpost.db"), nil
}

// Open creates the parent directory if needed and applies the schema. WAL keeps
// a long `ls` from blocking a concurrent `add`.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create %s: %w", dir, err)
		}
	}
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// Add inserts a note and returns it with the assigned id.
func (s *Store) Add(body, tag string) (Note, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return Note{}, errors.New("note body is empty")
	}
	n := Note{Body: body, Tag: strings.TrimSpace(tag), AddedAt: time.Now().UTC()}
	res, err := s.db.Exec(
		`INSERT INTO notes (body, tag, done, added_at) VALUES (?, ?, 0, ?)`,
		n.Body, n.Tag, n.AddedAt.Format(time.RFC3339),
	)
	if err != nil {
		return Note{}, fmt.Errorf("insert note: %w", err)
	}
	if n.ID, err = res.LastInsertId(); err != nil {
		return Note{}, fmt.Errorf("read inserted id: %w", err)
	}
	return n, nil
}

// List returns notes oldest first, open ones only unless all is set. A non-empty
// tag filters on an exact match.
func (s *Store) List(all bool, tag string) ([]Note, error) {
	q := `SELECT id, body, tag, done, added_at, closed_at FROM notes`
	var where []string
	var args []any
	if !all {
		where = append(where, "done = 0")
	}
	if tag = strings.TrimSpace(tag); tag != "" {
		where = append(where, "tag = ?")
		args = append(args, tag)
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY done, added_at, id"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var out []Note
	for rows.Next() {
		var (
			n      Note
			added  string
			closed sql.NullString
		)
		if err := rows.Scan(&n.ID, &n.Body, &n.Tag, &n.Done, &added, &closed); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		n.AddedAt, _ = time.Parse(time.RFC3339, added)
		if closed.Valid {
			n.ClosedAt, _ = time.Parse(time.RFC3339, closed.String)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// SetDone closes a note or reopens it. Closing a closed note is not an error.
func (s *Store) SetDone(id int64, done bool) error {
	var closed any
	if done {
		closed = time.Now().UTC().Format(time.RFC3339)
	}
	res, err := s.db.Exec(`UPDATE notes SET done = ?, closed_at = ? WHERE id = ?`, done, closed, id)
	if err != nil {
		return fmt.Errorf("update note %d: %w", id, err)
	}
	return checkAffected(res, id)
}

// Remove deletes a note outright.
func (s *Store) Remove(id int64) error {
	res, err := s.db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete note %d: %w", id, err)
	}
	return checkAffected(res, id)
}

// Purge deletes every closed note and reports how many went.
func (s *Store) Purge() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM notes WHERE done = 1`)
	if err != nil {
		return 0, fmt.Errorf("purge closed notes: %w", err)
	}
	return res.RowsAffected()
}

func checkAffected(res sql.Result, id int64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %d", ErrNotFound, id)
	}
	return nil
}
