package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
	"pharmacycounter/domain"
)

var (
	bucketOrders    = []byte("pickup_orders")
	bucketCalls     = []byte("call_records")
	bucketDispenses = []byte("dispense_records")
	bucketAudits    = []byte("audit_entries")
	bucketCounters  = []byte("counters")
	bucketIndexes   = []byte("indexes")
)

type Store struct {
	db   *bbolt.DB
	path string
}

type Transaction struct {
	tx *bbolt.Tx
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("数据库路径不能为空")
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, fmt.Errorf("创建数据库目录: %w", err)
	}
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}
	store := &Store{db: db, path: path}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) initialize() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		buckets := [][]byte{bucketOrders, bucketCalls, bucketDispenses, bucketAudits, bucketCounters, bucketIndexes}
		for _, name := range buckets {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return fmt.Errorf("创建数据桶 %s: %w", string(name), err)
			}
		}
		return nil
	})
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) View(fn func(*Transaction) error) error {
	if fn == nil {
		return errors.New("只读事务函数不能为空")
	}
	return s.db.View(func(tx *bbolt.Tx) error {
		return fn(&Transaction{tx: tx})
	})
}

func (s *Store) Update(fn func(*Transaction) error) error {
	if fn == nil {
		return errors.New("写事务函数不能为空")
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		return fn(&Transaction{tx: tx})
	})
}

func encode(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("编码记录: %w", err)
	}
	return data, nil
}

func decode(data []byte, target any) error {
	if len(data) == 0 {
		return domain.ErrNotFound
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("解码记录: %w", err)
	}
	return nil
}

func put(bucket *bbolt.Bucket, key string, value any) error {
	if key == "" {
		return errors.New("记录键不能为空")
	}
	data, err := encode(value)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), data)
}

func get(bucket *bbolt.Bucket, key string, target any) error {
	return decode(bucket.Get([]byte(key)), target)
}

func collect[T any](bucket *bbolt.Bucket) ([]T, error) {
	result := make([]T, 0)
	err := bucket.ForEach(func(_, value []byte) error {
		var item T
		if err := decode(value, &item); err != nil {
			return err
		}
		result = append(result, item)
		return nil
	})
	return result, err
}
