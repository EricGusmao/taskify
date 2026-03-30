package users

import "github.com/labstack/echo/v5"

// RegisterRoutes registers user routes on the provided group.
func RegisterRoutes(g *echo.Group, h *Handler) {
	g.POST("/me/avatar", h.UploadAvatar)
}
