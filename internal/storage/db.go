package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type Task struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Filename  string    `json:"filename"` // Added for UI display logic matching Frontend
	Path      string    `json:"path"`
	Status    string    `json:"status"` // downloading, completed, paused, error
	Progress  float64   `json:"progress"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Storage struct {
	db *badger.DB
}

func NewStorage() (*Storage, error) {
	appData, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(appData, "Tachyon", "data")

	// Ensure directory exists
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil // Disable default logger to avoid noise

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveTask(task Task) error {
	task.UpdatedAt = time.Now()

	bytes, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("task_"+task.ID), bytes)
	})
}

func (s *Storage) DeleteTask(id string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("task_" + id))
	})
}

func (s *Storage) GetAllTasks() ([]Task, error) {
	var tasks []Task

	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("task_")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				var task Task
				if err := json.Unmarshal(v, &task); err != nil {
					return err
				}
				tasks = append(tasks, task)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by CreatedAt descending (newest first)
	slices.SortFunc(tasks, func(a, b Task) int {
		if b.CreatedAt.Before(a.CreatedAt) {
			return -1
		}
		return 1
	})

	return tasks, nil
}
