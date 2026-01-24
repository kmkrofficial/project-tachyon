package storage

import (
	"encoding/binary"
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
	Category  string    `json:"category"`
	Priority  int       `json:"priority"` // 0=Low, 1=Normal, 2=High
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

func (s *Storage) GetTask(id string) (Task, error) {
	var task Task
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("task_" + id))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &task)
		})
	})
	return task, err
}

// IncrementStat atomically increments a counter
func (s *Storage) IncrementStat(key string, delta int64) error {
	return s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		var current int64
		if err == nil {
			item.Value(func(val []byte) error {
				current = int64(binary.BigEndian.Uint64(val)) // Simple storing as bytes
				// Actually Badger has MergeOperator for counters, but let's stick to simple Get/Set for now
				// Wait, JSON number is easier if we want interop, but binary is faster.
				// Let's use JSON for consistency with other data or just string conversion.
				// Reverting to JSON/String for simplicity.
				var valInt int64
				json.Unmarshal(val, &valInt)
				current = valInt
				return nil
			})
		} else if err != badger.ErrKeyNotFound {
			return err
		}

		current += delta
		valBytes, _ := json.Marshal(current)
		return txn.Set([]byte(key), valBytes)
	})
}

func (s *Storage) GetStatInt(key string) (int64, error) {
	var val int64
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			return json.Unmarshal(v, &val)
		})
	})
	if err == badger.ErrKeyNotFound {
		return 0, nil
	}
	return val, err
}

// GetStringList retrieves a list of strings from storage (e.g., domain whitelist/blacklist)
func (s *Storage) GetStringList(key string) ([]string, error) {
	var list []string
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			return json.Unmarshal(v, &list)
		})
	})
	if err == badger.ErrKeyNotFound {
		return []string{}, nil
	}
	return list, err
}

// SetStringList stores a list of strings in storage
func (s *Storage) SetStringList(key string, list []string) error {
	bytes, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), bytes)
	})
}

// GetString retrieves a single string value from storage
func (s *Storage) GetString(key string) (string, error) {
	var val string
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			val = string(v)
			return nil
		})
	})
	if err == badger.ErrKeyNotFound {
		return "", nil
	}
	return val, err
}

// SetString stores a single string value in storage
func (s *Storage) SetString(key string, val string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), []byte(val))
	})
}
