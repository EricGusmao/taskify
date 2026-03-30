package users_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/EricGusmao/taskify/internal/middleware"
	"github.com/EricGusmao/taskify/internal/testhelper"
	"github.com/EricGusmao/taskify/internal/testhelper/dbfactory"
	"github.com/EricGusmao/taskify/internal/users"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newTestEcho() *echo.Echo {
	return echo.New()
}

type handlerBundle struct {
	handler *users.Handler
	e       *echo.Echo
	storage *users.LocalStorage
	tx      *gorm.DB
}

func setupHandler(t *testing.T) (*handlerBundle, context.Context) {
	t.Helper()
	db := testhelper.NewMySQLContainer(t)
	tx := testhelper.TestTx(t, db)

	dir := t.TempDir()
	storage, err := users.NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	repo := users.NewRepository(tx)
	svc := users.NewService(repo, storage, zap.NewNop())
	handler := users.NewHandler(svc)

	return &handlerBundle{
		handler: handler,
		e:       newTestEcho(),
		storage: storage,
		tx:      tx,
	}, context.Background()
}

// buildMultipartRequest creates a multipart/form-data request with the given file content under field "avatar".
func buildMultipartRequest(t *testing.T, content []byte, filename string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("avatar", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/users/me/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func doUpload(t *testing.T, b *handlerBundle, req *http.Request, userID uint) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	c := b.e.NewContext(req, rec)
	c.Set(middleware.ContextKeyUserID, strconv.FormatUint(uint64(userID), 10))
	if err := b.handler.UploadAvatar(c); err != nil {
		b.e.HTTPErrorHandler(c, err)
	}
	return rec
}

// minimalJPEG returns bytes that http.DetectContentType identifies as image/jpeg.
func minimalJPEG() []byte {
	return append([]byte{0xff, 0xd8, 0xff, 0xe0}, make([]byte, 508)...)
}

// minimalPNG returns bytes that http.DetectContentType identifies as image/png.
func minimalPNG() []byte {
	return append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 504)...)
}

func TestHandler_UploadAvatar_ValidJPEG(t *testing.T) {
	t.Parallel()
	b, ctx := setupHandler(t)

	user := dbfactory.User(ctx, t, b.tx, nil)
	req := buildMultipartRequest(t, minimalJPEG(), "photo.jpg")

	rec := doUpload(t, b, req, user.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp users.UploadAvatarResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.AvatarURL == "" {
		t.Error("expected non-empty avatar_url")
	}
}

func TestHandler_UploadAvatar_ValidPNG(t *testing.T) {
	t.Parallel()
	b, ctx := setupHandler(t)

	user := dbfactory.User(ctx, t, b.tx, nil)
	req := buildMultipartRequest(t, minimalPNG(), "photo.png")

	rec := doUpload(t, b, req, user.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_UploadAvatar_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	b, ctx := setupHandler(t)

	user := dbfactory.User(ctx, t, b.tx, nil)
	req := buildMultipartRequest(t, []byte("GIF89a fake gif content"), "anim.gif")

	rec := doUpload(t, b, req, user.ID)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
}


func TestHandler_UploadAvatar_MissingFile(t *testing.T) {
	t.Parallel()
	b, ctx := setupHandler(t)

	user := dbfactory.User(ctx, t, b.tx, nil)
	req := httptest.NewRequest(http.MethodPost, "/users/me/avatar", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

	rec := doUpload(t, b, req, user.ID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_UploadAvatar_UpdatesDB(t *testing.T) {
	t.Parallel()
	b, ctx := setupHandler(t)

	user := dbfactory.User(ctx, t, b.tx, nil)
	req := buildMultipartRequest(t, minimalJPEG(), "photo.jpg")

	rec := doUpload(t, b, req, user.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var avatarURL string
	if err := b.tx.Table("users").
		Select("avatar_url").
		Where("id = ?", user.ID).
		Scan(&avatarURL).Error; err != nil {
		t.Fatalf("query avatar_url: %v", err)
	}
	if avatarURL == "" {
		t.Error("expected avatar_url to be set in DB")
	}
}

func TestHandler_UploadAvatar_MissingUserID(t *testing.T) {
	t.Parallel()
	b, _ := setupHandler(t)

	req := buildMultipartRequest(t, minimalJPEG(), "photo.jpg")
	rec := httptest.NewRecorder()
	c := b.e.NewContext(req, rec)
	// Deliberately omit setting middleware.ContextKeyUserID
	if err := b.handler.UploadAvatar(c); err != nil {
		b.e.HTTPErrorHandler(c, err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandler_UploadAvatar_ReplacesOldAvatar(t *testing.T) {
	t.Parallel()
	b, ctx := setupHandler(t)

	user := dbfactory.User(ctx, t, b.tx, nil)

	// First upload.
	req1 := buildMultipartRequest(t, minimalJPEG(), "first.jpg")
	rec1 := doUpload(t, b, req1, user.ID)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first upload: expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}
	var resp1 users.UploadAvatarResponse
	if err := json.NewDecoder(rec1.Body).Decode(&resp1); err != nil {
		t.Fatalf("decode first response: %v", err)
	}

	// Second upload.
	req2 := buildMultipartRequest(t, minimalPNG(), "second.png")
	rec2 := doUpload(t, b, req2, user.ID)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second upload: expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var resp2 users.UploadAvatarResponse
	if err := json.NewDecoder(rec2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode second response: %v", err)
	}

	if resp1.AvatarURL == resp2.AvatarURL {
		t.Errorf("expected different avatar URLs after replacement, got same: %q", resp1.AvatarURL)
	}

	// Verify DB holds the second URL.
	var avatarURL string
	if err := b.tx.Table("users").
		Select("avatar_url").
		Where("id = ?", user.ID).
		Scan(&avatarURL).Error; err != nil {
		t.Fatalf("query avatar_url: %v", err)
	}
	if avatarURL != resp2.AvatarURL {
		t.Errorf("expected DB avatar_url=%q, got %q", resp2.AvatarURL, avatarURL)
	}
	_ = fmt.Sprintf("replaced %s with %s", resp1.AvatarURL, resp2.AvatarURL)
}
