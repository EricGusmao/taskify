package auth

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
)

// Handler handles HTTP requests for the auth slice.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler with the provided service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register handles POST /auth/register.
func (h *Handler) Register(c *echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid request body")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}

	user, err := h.svc.Register(c.Request().Context(), RegisterInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			return echo.NewHTTPError(http.StatusConflict, "email already taken")
		}
		return err
	}

	return c.JSON(http.StatusCreated, RegisterResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Score: user.Score,
	})
}
