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

// MemberResponse is a single member entry in the list response.
type MemberResponse struct {
	UserID uint   `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Score  int    `json:"score"`
}

// ListMembersResponse is the paginated response for GET /teams/:id/members.
type ListMembersResponse struct {
	Data     []MemberResponse `json:"data"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Total    int64            `json:"total"`
}

// RankingEntryResponse is a single entry in the ranking response.
type RankingEntryResponse struct {
	Rank   int    `json:"rank"`
	UserID uint   `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Score  int    `json:"score"`
}

// RankingResponse is the paginated response for GET /teams/:id/ranking.
type RankingResponse struct {
	Data     []RankingEntryResponse `json:"data"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Total    int64                  `json:"total"`
}

// GetRanking handles GET /teams/:id/ranking.
func (h *Handler) GetRanking(c *echo.Context) error {
	rawID := c.Param("id")
	teamID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid team id")
	}

	page := 1
	if v, err := strconv.Atoi(c.QueryParam("page")); err == nil && v >= 1 {
		page = v
	}

	pageSize := 20
	if v, err := strconv.Atoi(c.QueryParam("page_size")); err == nil && v >= 1 && v <= 100 {
		pageSize = v
	}

	result, err := h.svc.GetRanking(c.Request().Context(), RankingInput{
		TeamID:   uint(teamID),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "team not found")
		}
		return err
	}

	data := make([]RankingEntryResponse, len(result.Data))
	for i, e := range result.Data {
		data[i] = RankingEntryResponse{
			Rank:   e.Rank,
			UserID: e.UserID,
			Name:   e.Name,
			Email:  e.Email,
			Score:  e.Score,
		}
	}

	return c.JSON(http.StatusOK, RankingResponse{
		Data:     data,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

// ListMembers handles GET /teams/:id/members.
func (h *Handler) ListMembers(c *echo.Context) error {
	rawID := c.Param("id")
	teamID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid team id")
	}

	page := 1
	if v, err := strconv.Atoi(c.QueryParam("page")); err == nil && v >= 1 {
		page = v
	}

	pageSize := 20
	if v, err := strconv.Atoi(c.QueryParam("page_size")); err == nil && v >= 1 && v <= 100 {
		pageSize = v
	}

	result, err := h.svc.ListMembers(c.Request().Context(), ListMembersInput{
		TeamID:   uint(teamID),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		if errors.Is(err, ErrTeamNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "team not found")
		}
		return err
	}

	data := make([]MemberResponse, len(result.Data))
	for i, m := range result.Data {
		data[i] = MemberResponse{
			UserID: m.UserID,
			Name:   m.Name,
			Email:  m.Email,
			Score:  m.Score,
		}
	}

	return c.JSON(http.StatusOK, ListMembersResponse{
		Data:     data,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
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
