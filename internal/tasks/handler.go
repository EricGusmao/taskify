package tasks

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/EricGusmao/taskify/internal/middleware"
	"github.com/labstack/echo/v5"
)

// Handler handles HTTP requests for the tasks slice.
type Handler struct {
	svc *Service
}

// NewHandler returns a Handler with the provided service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// CreateRequest is the payload for POST /teams/:id/tasks.
type CreateRequest struct {
	Title  string `json:"title"  validate:"required,min=1,max=255"`
	Points int    `json:"points" validate:"required,gt=0"`
}

// CreateResponse is returned on successful task creation.
type CreateResponse struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Points int    `json:"points"`
	TeamID uint   `json:"team_id"`
}

// Create handles POST /teams/:id/tasks.
func (h *Handler) Create(c *echo.Context) error {
	rawID := c.Param("id")
	teamID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid team id")
	}

	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "invalid request body")
	}
	if err := c.Validate(req); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}

	task, err := h.svc.Create(c.Request().Context(), CreateInput{
		TeamID: uint(teamID),
		Title:  req.Title,
		Points: req.Points,
	})
	if err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "team not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusCreated, CreateResponse{
		ID:     task.ID,
		Title:  task.Title,
		Points: task.Points,
		TeamID: task.TeamID,
	})
}

// CompleteResponse is returned on successful task completion.
type CompleteResponse struct {
	ID           uint       `json:"id"`
	Title        string     `json:"title"`
	Points       int        `json:"points"`
	TeamID       uint       `json:"team_id"`
	DoneByUserID *uint      `json:"done_by_user_id"`
	DoneAt       *time.Time `json:"done_at"`
}

// Complete handles PATCH /tasks/:id/complete.
func (h *Handler) Complete(c *echo.Context) error {
	rawTaskID := c.Param("id")
	taskID, err := strconv.ParseUint(rawTaskID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
	}

	rawUserID, ok := c.Get(middleware.ContextKeyUserID).(string)
	if !ok || rawUserID == "" {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	userID, err := strconv.ParseUint(rawUserID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	task, err := h.svc.Complete(c.Request().Context(), uint(taskID), uint(userID))
	if err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "task not found")
		case errors.Is(err, ErrAlreadyCompleted):
			return echo.NewHTTPError(http.StatusConflict, "task already completed")
		case errors.Is(err, ErrNotMember):
			return echo.NewHTTPError(http.StatusForbidden, "forbidden")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, CompleteResponse{
		ID:           task.ID,
		Title:        task.Title,
		Points:       task.Points,
		TeamID:       task.TeamID,
		DoneByUserID: task.DoneByUserID,
		DoneAt:       task.DoneAt,
	})
}
