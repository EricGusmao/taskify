// Package docs serves the OpenAPI specification and Redocly UI.
package docs

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// Handler serves the OpenAPI spec and the Redocly documentation UI.
type Handler struct {
	spec []byte
}

// NewHandler returns a Handler that serves the provided OpenAPI spec bytes.
func NewHandler(spec []byte) *Handler {
	return &Handler{spec: spec}
}

// ServeSpec serves the raw OpenAPI YAML spec at GET /api/openapi.yaml.
func (h *Handler) ServeSpec(c *echo.Context) error {
	return c.Blob(http.StatusOK, "application/yaml", h.spec)
}

// ServeRedoc serves the Redocly HTML documentation page at GET /docs.
func (h *Handler) ServeRedoc(c *echo.Context) error {
	const html = `<!DOCTYPE html>
<html>
<head>
  <title>Taskify API</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>body { margin: 0; padding: 0; }</style>
</head>
<body>
  <redoc spec-url='/api/openapi.yaml'></redoc>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`
	return c.HTML(http.StatusOK, html)
}
