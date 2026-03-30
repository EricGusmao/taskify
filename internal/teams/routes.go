package teams

import "github.com/labstack/echo/v5"

// RegisterRoutes registers teams routes on the provided group.
func RegisterRoutes(g *echo.Group, h *Handler) {
	g.POST("", h.Create)
	g.POST("/:id/members", h.AddMember)
}
