# tickpost

Sticky notes and todos that stay in your terminal, kept in a single SQLite file.

Every note app I tried wanted an account and a sync service. What I actually
wanted was to type one line before a deploy and read it back twenty minutes
later. No server, no daemon, no config file: one binary and one database file
you can copy, back up or delete.

## Install

```
go install github.com/arvelquigley99/tickpost@latest
```

Requires Go 1.22 or newer. SQLite is compiled in via `modernc.org/sqlite`, so
there is no cgo and no system SQLite to install.

## Usage

```
tickpost add <text> [-tag NAME]   add a note
tickpost ls [-a] [-tag NAME]      list open notes, -a includes closed ones
tickpost done <id>                close a note
tickpost open <id>                reopen a closed note
tickpost rm <id>                  delete a note
tickpost purge                    delete every closed note
```

A normal session:

```
$ tickpost add "drain the queue before deploy" -tag ops
added #1
$ tickpost add -tag pg "reindex orders_created_idx concurrently"
added #2
$ tickpost ls
ID  AGE       TAG  NOTE
1   just now  ops  drain the queue before deploy
2   just now  pg   reindex orders_created_idx concurrently
$ tickpost done 1
closed #1
```

The `-tag` flag works before or after the note text, because remembering an
argument order is exactly the kind of friction this was meant to avoid.

## Where the notes live

`$XDG_DATA_HOME/tickpost/tickpost.db`, falling back to
`~/.local/share/tickpost/tickpost.db`. Set `TICKPOST_DB` to point somewhere
else, which is useful for per-project note files:

```
export TICKPOST_DB=./notes.db
```

## Licence

MIT. See [LICENSE](LICENSE).
