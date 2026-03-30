package teams

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
)

// Handler handles HTTP requests for the teams slice.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler with the provided service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// CreateRequest is the payload for POST /teams.
type CreateRequest struct {
	Name string `json:"name" validate:"required,min=1,max=255"`
}

// CreateResponse is returned on successful team creation.
type CreateResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// Create handles POST /teams.
func (h *Handler) Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid request body")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}

	team, err := h.svc.Create(c.Request().Context(), CreateInput{
		Name: req.Name,
	})
	if err != nil {
		if errors.Is(err, ErrNameTaken) {
			return echo.NewHTTPError(http.StatusConflict, "team name already taken")
		}
		return err
	}

	return c.JSON(http.StatusCreated, CreateResponse{
		ID:   team.ID,
		Name: team.Name,
	})
}
