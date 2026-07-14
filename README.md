# muhammed

A simple command-line task manager written in Go.

## Build

```sh
go build -o todo .
```

## Usage

```sh
todo add <text>      # Add a new task
todo list [filter]   # List tasks (filter: --all, --done, --pending; default: --pending)
todo done <id>       # Mark a task as complete
todo delete <id>     # Delete a task
todo help            # Show help
```

Tasks are stored in `~/.muhammed-todo/tasks.json`.
