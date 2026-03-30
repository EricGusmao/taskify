package auth

import "github.com/labstack/echo/v5"

// RegisterRoutes registers auth routes on the provided group.
func RegisterRoutes(g *echo.Group, h *Handler) {
	g.POST("/register", h.Register)
}
