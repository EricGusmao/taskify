package users

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	_ "golang.org/x/image/webp"
	"go.uber.org/zap"
)

// allowedFormats maps image.DecodeConfig format names to file extensions.
// WebP is re-encoded as JPEG on upload (no Go WebP encoder exists).
var allowedFormats = map[string]string{
	"jpeg": ".jpg",
	"png":  ".png",
	"webp": ".jpg", // re-encoded to JPEG
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

// maxPixels is the maximum allowed image area (16 MP) to prevent decompression bombs.
const maxPixels = 4096 * 4096

// UploadAvatar validates the file, stores it, and updates the user's avatar_url.
// Content is validated by fully parsing the image header via image.DecodeConfig,
// which rejects polyglot files that merely spoof magic bytes.
// Returns the stored path/URL.
func (s *Service) UploadAvatar(ctx context.Context, userID uint, content io.ReadSeeker) (string, error) {
	reencoded, ext, err := sanitizeImage(content)
	if err != nil {
		return "", fmt.Errorf("users.service.UploadAvatar: %w", err)
	}

	fname := fmt.Sprintf("%d%s", userID, ext)

	if err := s.replaceAvatar(ctx, userID); err != nil {
		return "", fmt.Errorf("users.service.UploadAvatar: %w", err)
	}

	url, err := s.storage.Upload(ctx, fname, reencoded)
	if err != nil {
		return "", fmt.Errorf("users.service.UploadAvatar: upload: %w", err)
	}

	if err := s.repo.UpdateAvatarURL(ctx, userID, url); err != nil {
		return "", fmt.Errorf("users.service.UploadAvatar: %w", err)
	}

	s.logger.Debug("avatar uploaded", zap.Uint("user_id", userID), zap.String("url", url))
	return url, nil
}

// sanitizeImage validates the image format and dimensions, then re-encodes it to strip
// any embedded metadata or payloads. Returns the re-encoded bytes and the file extension.
func sanitizeImage(content io.ReadSeeker) (*bytes.Buffer, string, error) {
	cfg, format, err := image.DecodeConfig(content)
	if err != nil {
		return nil, "", ErrUnsupportedFormat
	}
	ext, ok := allowedFormats[format]
	if !ok {
		return nil, "", ErrUnsupportedFormat
	}
	if cfg.Width*cfg.Height > maxPixels {
		return nil, "", ErrImageTooLarge
	}

	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return nil, "", fmt.Errorf("seek: %w", err)
	}

	img, _, err := image.Decode(content)
	if err != nil {
		return nil, "", ErrUnsupportedFormat
	}

	var out bytes.Buffer
	switch format {
	case "png":
		if err := png.Encode(&out, img); err != nil {
			return nil, "", fmt.Errorf("reencode png: %w", err)
		}
	default: // jpeg and webp → jpeg
		if err := jpeg.Encode(&out, img, nil); err != nil {
			return nil, "", fmt.Errorf("reencode jpeg: %w", err)
		}
	}

	return &out, ext, nil
}

// replaceAvatar deletes the user's existing avatar from storage, if any.
func (s *Service) replaceAvatar(ctx context.Context, userID uint) error {
	oldURL, err := s.repo.GetAvatarURL(ctx, userID)
	if err != nil {
		s.logger.Warn("users.service.UploadAvatar: get old avatar url", zap.Error(err))
		return nil
	}
	if oldURL == "" {
		return nil
	}
	if err := s.storage.Delete(ctx, oldURL); err != nil {
		s.logger.Warn("users.service.UploadAvatar: delete old avatar", zap.String("path", oldURL), zap.Error(err))
	}
	return nil
}
