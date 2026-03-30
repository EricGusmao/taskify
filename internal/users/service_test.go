package users_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/EricGusmao/taskify/internal/users"
	"go.uber.org/zap"
)

// mockStorage is a test double for StorageProvider.
type mockStorage struct {
	uploadFn func(ctx context.Context, filename string, content io.Reader) (string, error)
	deleteFn func(ctx context.Context, filename string) error
	deleted  []string
}

func (m *mockStorage) Upload(ctx context.Context, filename string, content io.Reader) (string, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, filename, content)
	}
	// Default: drain reader, return filename as url.
	_, _ = io.Copy(io.Discard, content)
	return "/uploads/" + filename, nil
}

func (m *mockStorage) Delete(ctx context.Context, filename string) error {
	m.deleted = append(m.deleted, filename)
	if m.deleteFn != nil {
		return m.deleteFn(ctx, filename)
	}
	return nil
}

// mockUserRepo is a test double for UserRepository.
type mockUserRepo struct {
	avatarURL    string
	updatedURL   string
	getErr       error
	updateErr    error
}

func (m *mockUserRepo) GetAvatarURL(_ context.Context, _ uint) (string, error) {
	return m.avatarURL, m.getErr
}

func (m *mockUserRepo) UpdateAvatarURL(_ context.Context, _ uint, url string) error {
	m.updatedURL = url
	return m.updateErr
}

func TestService_UploadAvatar_RejectsNonImage(t *testing.T) {
	t.Parallel()
	repo := &mockUserRepo{}
	storage := &mockStorage{}
	svc := users.NewService(repo, storage, zap.NewNop())

	_, err := svc.UploadAvatar(context.Background(), 1, strings.NewReader("plain text content"))
	if !errors.Is(err, users.ErrUnsupportedFormat) {
		t.Errorf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestService_UploadAvatar_AcceptsJPEG(t *testing.T) {
	t.Parallel()
	repo := &mockUserRepo{}
	storage := &mockStorage{}
	svc := users.NewService(repo, storage, zap.NewNop())

	// Minimal JPEG: SOI marker + padding
	jpeg := append([]byte{0xff, 0xd8, 0xff, 0xe0}, make([]byte, 508)...)

	url, err := svc.UploadAvatar(context.Background(), 1, strings.NewReader(string(jpeg)))
	if err != nil {
		t.Fatalf("expected no error for JPEG, got %v", err)
	}
	if url == "" {
		t.Error("expected non-empty url")
	}
}

func TestService_UploadAvatar_AcceptsPNG(t *testing.T) {
	t.Parallel()
	repo := &mockUserRepo{}
	storage := &mockStorage{}
	svc := users.NewService(repo, storage, zap.NewNop())

	// PNG magic bytes
	png := append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 504)...)

	url, err := svc.UploadAvatar(context.Background(), 1, strings.NewReader(string(png)))
	if err != nil {
		t.Fatalf("expected no error for PNG, got %v", err)
	}
	if url == "" {
		t.Error("expected non-empty url")
	}
}

func TestService_UploadAvatar_AcceptsWebP(t *testing.T) {
	t.Parallel()
	repo := &mockUserRepo{}
	storage := &mockStorage{}
	svc := users.NewService(repo, storage, zap.NewNop())

	// WebP: RIFF....WEBP
	webp := []byte("RIFF\x00\x00\x00\x00WEBPVP")
	webp = append(webp, make([]byte, 500)...)

	url, err := svc.UploadAvatar(context.Background(), 1, strings.NewReader(string(webp)))
	if err != nil {
		t.Fatalf("expected no error for WebP, got %v", err)
	}
	if url == "" {
		t.Error("expected non-empty url")
	}
}

func TestService_UploadAvatar_DeletesOldAvatar(t *testing.T) {
	t.Parallel()
	repo := &mockUserRepo{avatarURL: "/uploads/old.jpg"}
	storage := &mockStorage{}
	svc := users.NewService(repo, storage, zap.NewNop())

	jpeg := append([]byte{0xff, 0xd8, 0xff, 0xe0}, make([]byte, 508)...)
	_, err := svc.UploadAvatar(context.Background(), 1, strings.NewReader(string(jpeg)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(storage.deleted) == 0 {
		t.Fatal("expected old avatar to be deleted")
	}
	if storage.deleted[0] != "/uploads/old.jpg" {
		t.Errorf("expected deletion of /uploads/old.jpg, got %q", storage.deleted[0])
	}
}

func TestService_UploadAvatar_PropagatesStorageError(t *testing.T) {
	t.Parallel()
	repo := &mockUserRepo{}
	storageErr := errors.New("disk full")
	storage := &mockStorage{
		uploadFn: func(_ context.Context, _ string, _ io.Reader) (string, error) {
			return "", storageErr
		},
	}
	svc := users.NewService(repo, storage, zap.NewNop())

	jpeg := append([]byte{0xff, 0xd8, 0xff, 0xe0}, make([]byte, 508)...)
	_, err := svc.UploadAvatar(context.Background(), 1, strings.NewReader(string(jpeg)))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, storageErr) {
		t.Errorf("expected storage error to be wrapped, got %v", err)
	}
}
