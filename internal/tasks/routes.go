package tasks

import "github.com/labstack/echo/v5"

// RegisterRoutes registers task routes on the provided Echo group.
// g is expected to be the /teams group so that routes resolve as /teams/:id/tasks.
func RegisterRoutes(g *echo.Group, h *Handler) {
	g.POST("/:id/tasks", h.Create)
}

// RegisterTaskRoutes registers routes on the /tasks group.
func RegisterTaskRoutes(g *echo.Group, h *Handler) {
	g.PATCH("/:id/complete", h.Complete)
}
