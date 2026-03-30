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

// RegisterRequest is the payload for POST /auth/register.
type RegisterRequest struct {
	Name     string `json:"name"     validate:"required,min=1,max=255"`
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

// RegisterResponse is returned on successful registration.
type RegisterResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Score int    `json:"score"`
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

// LoginRequest is the payload for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse is returned on successful login.
type LoginResponse struct {
	Token string `json:"token"`
}

// Login handles POST /auth/login.
func (h *Handler) Login(c *echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid request body")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}

	token, err := h.svc.Login(c.Request().Context(), LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		return err
	}

	return c.JSON(http.StatusOK, LoginResponse{Token: token})
}
