package users_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EricGusmao/taskify/internal/users"
)

func TestLocalStorage_Upload(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	storage, err := users.NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	content := "hello world"
	path, err := storage.Upload(context.Background(), "test.txt", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Errorf("expected %q, got %q", content, string(got))
	}
}

func TestLocalStorage_Upload_ReturnsStoredPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	storage, err := users.NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	path, err := storage.Upload(context.Background(), "avatar.jpg", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	expected := filepath.Join(dir, "avatar.jpg")
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	storage, err := users.NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	path, err := storage.Upload(context.Background(), "to_delete.jpg", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	if err := storage.Delete(context.Background(), path); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted")
	}
}

func TestLocalStorage_Delete_NonExistentFileIsNotError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	storage, err := users.NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	if err := storage.Delete(context.Background(), filepath.Join(dir, "nonexistent.jpg")); err != nil {
		t.Errorf("expected no error deleting nonexistent file, got: %v", err)
	}
}
