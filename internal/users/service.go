package users

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// allowedContentTypes maps detected MIME types to file extensions.
var allowedContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// Service handles business logic for the users slice.
type Service struct {
	repo    UserRepository
	storage StorageProvider
	logger  *zap.Logger
}

// NewService returns a Service with the provided dependencies.
func NewService(repo UserRepository, storage StorageProvider, logger *zap.Logger) *Service {
	return &Service{repo: repo, storage: storage, logger: logger}
}

// UploadAvatar validates the file, stores it, and updates the user's avatar_url.
// filename is used only for generating the stored filename extension fallback; content
// is always sniffed via http.DetectContentType for security.
// Returns the stored path/URL.
func (s *Service) UploadAvatar(ctx context.Context, userID uint, content io.Reader) (string, error) {
	// Read first 512 bytes for content-type detection without consuming the reader.
	buf := make([]byte, 512)
	n, err := io.ReadFull(content, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("users.service.UploadAvatar: read header: %w", err)
	}
	buf = buf[:n]

	ct := detectContentType(buf)
	ext, ok := allowedContentTypes[ct]
	if !ok {
		return "", fmt.Errorf("users.service.UploadAvatar: %w", ErrUnsupportedFormat)
	}

	// Reassemble the full reader.
	full := io.MultiReader(bytes.NewReader(buf), content)

	// Generate a safe filename.
	fname := fmt.Sprintf("%d%s", userID, ext)

	// Fetch and delete the old avatar (best-effort).
	oldURL, err := s.repo.GetAvatarURL(ctx, userID)
	if err != nil {
		s.logger.Warn("users.service.UploadAvatar: get old avatar url", zap.Error(err))
	} else if oldURL != "" {
		if delErr := s.storage.Delete(ctx, oldURL); delErr != nil {
			s.logger.Warn("users.service.UploadAvatar: delete old avatar", zap.String("path", oldURL), zap.Error(delErr))
		}
	}

	url, err := s.storage.Upload(ctx, fname, full)
	if err != nil {
		return "", fmt.Errorf("users.service.UploadAvatar: upload: %w", err)
	}

	if err := s.repo.UpdateAvatarURL(ctx, userID, url); err != nil {
		return "", fmt.Errorf("users.service.UploadAvatar: %w", err)
	}

	s.logger.Debug("avatar uploaded", zap.Uint("user_id", userID), zap.String("url", url))
	return url, nil
}

// detectContentType checks for WebP (which stdlib doesn't detect) before
// falling back to http.DetectContentType.
func detectContentType(buf []byte) string {
	// WebP: bytes 0-3 = "RIFF", bytes 8-11 = "WEBP"
	if len(buf) >= 12 &&
		string(buf[0:4]) == "RIFF" &&
		string(buf[8:12]) == "WEBP" {
		return "image/webp"
	}
	return http.DetectContentType(buf)
}

