package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Task struct {
	ID          int        `json:"id"`
	Text        string     `json:"text"`
	Done        bool       `json:"done"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Store struct {
	path string
}

func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}
	dir := filepath.Join(home, ".muhammed-todo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}
	return &Store{path: filepath.Join(dir, "tasks.json")}, nil
}

func (s *Store) Load() ([]Task, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []Task{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading tasks file: %w", err)
	}
	if len(data) == 0 {
		return []Task{}, nil
	}
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parsing tasks file: %w", err)
	}
	return tasks, nil
}

func (s *Store) Save(tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding tasks: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("writing tasks file: %w", err)
	}
	return nil
}

func nextID(tasks []Task) int {
	max := 0
	for _, t := range tasks {
		if t.ID > max {
			max = t.ID
		}
	}
	return max + 1
}
