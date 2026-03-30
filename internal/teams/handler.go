package teams

import (
	"errors"
	"net/http"
	"strconv"

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

// AddMemberRequest is the payload for POST /teams/:id/members.
type AddMemberRequest struct {
	UserID uint `json:"user_id" validate:"required,gt=0"`
}

// AddMemberResponse is returned on successful member addition.
type AddMemberResponse struct {
	TeamID uint `json:"team_id"`
	UserID uint `json:"user_id"`
}

// AddMember handles POST /teams/:id/members.
func (h *Handler) AddMember(c *echo.Context) error {
	rawID := c.Param("id")
	teamID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid team id")
	}

	var req AddMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid request body")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}

	member, err := h.svc.AddMember(c.Request().Context(), AddMemberInput{
		TeamID: uint(teamID),
		UserID: req.UserID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrTeamNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "team not found")
		case errors.Is(err, ErrUserNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		case errors.Is(err, ErrAlreadyMember):
			return echo.NewHTTPError(http.StatusConflict, "user is already a member of this team")
		}
		return err
	}

	return c.JSON(http.StatusCreated, AddMemberResponse{
		TeamID: member.TeamID,
		UserID: member.UserID,
	})
}
