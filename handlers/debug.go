package handlers

import (
	"net/http"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/mapcache"
	"github.com/labstack/echo/v4"
)

type DebugBuildInfo struct {
	BuildInfo string
}

type DebugSessions struct {
	Sessions []int `json:"sessions"`
}

type Debug struct {
	UserCache *mapcache.Map[cohabitaters.UserState]
}

func (h *Debug) BuildInfo(c echo.Context) error {
	return c.JSON(http.StatusOK, struct{ BuildInfo string }{cohabitaters.BuildInfo()})
}

func (h *Debug) Sessions(c echo.Context) error {
	m := []int{}
	h.UserCache.Range(func(id int, t cohabitaters.UserState) bool {
		m = append(m, id)
		return true
	})
	return c.JSON(http.StatusOK, DebugSessions{Sessions: m})
}
