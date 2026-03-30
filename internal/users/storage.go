package users

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// StorageProvider abstracts file storage operations.
type StorageProvider interface {
	Upload(ctx context.Context, filename string, content io.Reader) (string, error)
	Delete(ctx context.Context, filename string) error
}

// LocalStorage implements StorageProvider by saving files to local disk.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage returns a LocalStorage that stores files under basePath.
// It creates basePath (and any parent directories) if they do not exist.
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("users.LocalStorage: create base dir: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

// Upload writes content to basePath/filename and returns the relative path.
func (s *LocalStorage) Upload(_ context.Context, filename string, content io.Reader) (string, error) {
	dst := filepath.Join(s.basePath, filename)
	f, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("users.LocalStorage.Upload: create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, content); err != nil {
		return "", fmt.Errorf("users.LocalStorage.Upload: write file: %w", err)
	}

	return dst, nil
}

// Delete removes filename from basePath. A non-existent file is not an error.
func (s *LocalStorage) Delete(_ context.Context, filename string) error {
	err := os.Remove(filename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("users.LocalStorage.Delete: %w", err)
	}
	return nil
}
