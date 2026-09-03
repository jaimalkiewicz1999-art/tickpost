// Command tickpost is a small sticky-note and todo keeper for the terminal.
// Notes live in a single SQLite file, so there is no server, no account and
// nothing to keep running in the background.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

const version = "0.3.1"

const usage = `tickpost - sticky notes that stay in your terminal

usage:
  tickpost add <text> [-tag NAME]   add a note
  tickpost ls [-a] [-tag NAME]      list open notes (-a includes closed ones)
  tickpost done <id>                close a note
  tickpost open <id>                reopen a closed note
  tickpost rm <id>                  delete a note
  tickpost purge                    delete every closed note
  tickpost version                  print the version

The database defaults to $XDG_DATA_HOME/tickpost/tickpost.db and can be
overridden with TICKPOST_DB, which is handy for per-project note files.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tickpost:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "version", "--version":
		fmt.Println("tickpost", version)
		return nil
	}

	path, err := DefaultPath()
	if err != nil {
		return err
	}
	store, err := Open(path)
	if err != nil {
		return err
	}
	defer store.Close()

	switch cmd {
	case "add":
		return cmdAdd(store, rest)
	case "ls", "list":
		return cmdList(store, rest)
	case "done":
		return cmdByID(rest, "closed", func(id int64) error { return store.SetDone(id, true) })
	case "open", "undone":
		return cmdByID(rest, "reopened", func(id int64) error { return store.SetDone(id, false) })
	case "rm", "remove":
		return cmdByID(rest, "deleted", store.Remove)
	case "purge":
		n, err := store.Purge()
		if err != nil {
			return err
		}
		fmt.Printf("removed %d closed note(s)\n", n)
		return nil
	default:
		return fmt.Errorf("unknown command %q, try 'tickpost help'", cmd)
	}
}

func cmdAdd(s *Store, args []string) error {
	tag, words, err := splitTag(args)
	if err != nil {
		return err
	}
	body := strings.Join(words, " ")
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("nothing to add, try: tickpost add \"drain the queue before deploy\"")
	}
	n, err := s.Add(body, tag)
	if err != nil {
		return err
	}
	fmt.Printf("added #%d\n", n.ID)
	return nil
}

// splitTag pulls -tag out of the arguments wherever it appears and returns the
// rest as the note body. The flag package stops at the first positional, which
// would quietly swallow "-tag ops" into the body of `add "ship it" -tag ops`.
func splitTag(args []string) (tag string, rest []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-tag" || arg == "--tag":
			if i+1 >= len(args) {
				return "", nil, errors.New("-tag needs a value")
			}
			tag = args[i+1]
			i++
		case strings.HasPrefix(arg, "-tag="), strings.HasPrefix(arg, "--tag="):
			tag = arg[strings.Index(arg, "=")+1:]
		default:
			rest = append(rest, arg)
		}
	}
	return tag, rest, nil
}

func cmdList(s *Store, args []string) error {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	all := fs.Bool("a", false, "include closed notes")
	tag := fs.String("tag", "", "only show notes with this tag")
	if err := fs.Parse(args); err != nil {
		return err
	}
	notes, err := s.List(*all, *tag)
	if err != nil {
		return err
	}
	if len(notes) == 0 {
		fmt.Println("nothing open. enjoy it while it lasts.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tAGE\tTAG\tNOTE")
	for _, n := range notes {
		mark := ""
		if n.Done {
			mark = "~ "
		}
		tagCol := n.Tag
		if tagCol == "" {
			tagCol = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s%s\n", n.ID, age(n.AddedAt), tagCol, mark, n.Body)
	}
	return w.Flush()
}

func cmdByID(args []string, verb string, apply func(int64) error) error {
	id, err := parseID(args)
	if err != nil {
		return err
	}
	if err := apply(id); err != nil {
		return err
	}
	fmt.Printf("%s #%d\n", verb, id)
	return nil
}

func parseID(args []string) (int64, error) {
	if len(args) != 1 {
		return 0, fmt.Errorf("expected exactly one note id")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%q is not a note id", args[0])
	}
	return id, nil
}

// age renders a duration the way you would say it out loud.
func age(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
