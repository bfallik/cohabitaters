package handlers

import (
	"net/http"

	"github.com/bfallik/cohabitaters"
	"github.com/labstack/echo/v4"
)

type DebugBuildInfo struct {
	BuildInfo string
}

type Debug struct{}

func (h *Debug) BuildInfo(c echo.Context) error {
	return c.JSON(http.StatusOK, struct{ BuildInfo string }{cohabitaters.BuildInfo()})
}
