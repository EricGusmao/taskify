package users

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/EricGusmao/taskify/internal/middleware"
	"github.com/labstack/echo/v5"
)

// Handler handles HTTP requests for the users slice.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler with the provided service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// UploadAvatarResponse is returned on successful avatar upload.
type UploadAvatarResponse struct {
	AvatarURL string `json:"avatar_url"`
}

// UploadAvatar handles POST /users/me/avatar.
func (h *Handler) UploadAvatar(c *echo.Context) error {
	rawUserID, ok := c.Get(middleware.ContextKeyUserID).(string)
	if !ok || rawUserID == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing user id")
	}
	uid, err := strconv.ParseUint(rawUserID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid user id")
	}

	_, file, err := c.Request().FormFile("avatar")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "avatar file is required")
	}

	src, err := file.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to open uploaded file")
	}
	defer src.Close()

	url, err := h.svc.UploadAvatar(c.Request().Context(), uint(uid), src)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedFormat):
			return echo.NewHTTPError(http.StatusUnprocessableEntity, "unsupported file format: only jpeg, png, and webp are allowed")
		case errors.Is(err, ErrUserNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}

	return c.JSON(http.StatusOK, UploadAvatarResponse{AvatarURL: url})
}
