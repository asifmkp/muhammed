package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	store, err := NewStore()
	if err != nil {
		fatal(err)
	}

	switch os.Args[1] {
	case "add":
		cmdAdd(store, os.Args[2:])
	case "list", "ls":
		cmdList(store, os.Args[2:])
	case "done", "complete":
		cmdDone(store, os.Args[2:])
	case "delete", "rm":
		cmdDelete(store, os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func cmdAdd(store *Store, args []string) {
	if len(args) == 0 {
		fatal(fmt.Errorf("usage: todo add <task text>"))
	}
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		fatal(fmt.Errorf("task text cannot be empty"))
	}

	tasks, err := store.Load()
	if err != nil {
		fatal(err)
	}

	task := Task{
		ID:        nextID(tasks),
		Text:      text,
		CreatedAt: time.Now(),
	}
	tasks = append(tasks, task)

	if err := store.Save(tasks); err != nil {
		fatal(err)
	}
	fmt.Printf("Added task #%d: %s\n", task.ID, task.Text)
}

func cmdList(store *Store, args []string) {
	filter := "pending"
	if len(args) > 0 {
		switch args[0] {
		case "--all", "-a", "all":
			filter = "all"
		case "--done", "-d", "done":
			filter = "done"
		case "--pending", "-p", "pending":
			filter = "pending"
		default:
			fatal(fmt.Errorf("unknown list filter: %s (use --all, --done, or --pending)", args[0]))
		}
	}

	tasks, err := store.Load()
	if err != nil {
		fatal(err)
	}

	var shown []Task
	for _, t := range tasks {
		switch filter {
		case "done":
			if t.Done {
				shown = append(shown, t)
			}
		case "pending":
			if !t.Done {
				shown = append(shown, t)
			}
		default:
			shown = append(shown, t)
		}
	}

	if len(shown) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	for _, t := range shown {
		box := "[ ]"
		if t.Done {
			box = "[x]"
		}
		fmt.Printf("%s #%d  %s\n", box, t.ID, t.Text)
	}
}

func cmdDone(store *Store, args []string) {
	id := parseID(args, "todo done <id>")

	tasks, err := store.Load()
	if err != nil {
		fatal(err)
	}

	found := false
	for i := range tasks {
		if tasks[i].ID == id {
			tasks[i].Done = true
			ts := time.Now()
			tasks[i].CompletedAt = &ts
			found = true
			break
		}
	}
	if !found {
		fatal(fmt.Errorf("no task with id %d", id))
	}

	if err := store.Save(tasks); err != nil {
		fatal(err)
	}
	fmt.Printf("Completed task #%d\n", id)
}

func cmdDelete(store *Store, args []string) {
	id := parseID(args, "todo delete <id>")

	tasks, err := store.Load()
	if err != nil {
		fatal(err)
	}

	idx := -1
	for i, t := range tasks {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		fatal(fmt.Errorf("no task with id %d", id))
	}
	tasks = append(tasks[:idx], tasks[idx+1:]...)

	if err := store.Save(tasks); err != nil {
		fatal(err)
	}
	fmt.Printf("Deleted task #%d\n", id)
}

func parseID(args []string, usage string) int {
	if len(args) == 0 {
		fatal(fmt.Errorf("usage: %s", usage))
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		fatal(fmt.Errorf("invalid task id: %s", args[0]))
	}
	return id
}

func printUsage() {
	fmt.Print(`todo - a simple task manager

Usage:
  todo add <text>         Add a new task
  todo list [filter]      List tasks (filter: --all, --done, --pending; default: --pending)
  todo done <id>          Mark a task as complete
  todo delete <id>        Delete a task
  todo help               Show this help message

Tasks are stored in ~/.muhammed-todo/tasks.json
`)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
