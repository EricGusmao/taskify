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

// TaskItemResponse is a single task entry in the list response.
type TaskItemResponse struct {
	ID           uint       `json:"id"`
	Title        string     `json:"title"`
	Points       int        `json:"points"`
	TeamID       uint       `json:"team_id"`
	DoneByUserID *uint      `json:"done_by_user_id"`
	DoneAt       *time.Time `json:"done_at"`
}

// ListByTeamResponse is the paginated response for GET /teams/:id/tasks.
type ListByTeamResponse struct {
	Data     []TaskItemResponse `json:"data"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Total    int64              `json:"total"`
}

// ListByTeam handles GET /teams/:id/tasks.
func (h *Handler) ListByTeam(c *echo.Context) error {
	rawID := c.Param("id")
	teamID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid team id")
	}

	var done *bool
	if v := c.QueryParam("done"); v == "true" {
		t := true
		done = &t
	} else if v == "false" {
		f := false
		done = &f
	}

	page := 1
	if v, err := strconv.Atoi(c.QueryParam("page")); err == nil && v >= 1 {
		page = v
	}

	pageSize := 20
	if v, err := strconv.Atoi(c.QueryParam("page_size")); err == nil && v >= 1 && v <= 100 {
		pageSize = v
	}

	result, err := h.svc.ListByTeam(c.Request().Context(), ListByTeamInput{
		TeamID:   uint(teamID),
		Done:     done,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "team not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	data := make([]TaskItemResponse, len(result.Data))
	for i, task := range result.Data {
		data[i] = TaskItemResponse{
			ID:           task.ID,
			Title:        task.Title,
			Points:       task.Points,
			TeamID:       task.TeamID,
			DoneByUserID: task.DoneByUserID,
			DoneAt:       task.DoneAt,
		}
	}

	return c.JSON(http.StatusOK, ListByTeamResponse{
		Data:     data,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
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
