package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestDebugBuildinfo(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/debug/buildinfo", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &Debug{}

	if err := h.BuildInfo(c); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if http.StatusOK != rec.Code {
		t.Errorf("expected:200, got: %v", rec.Code)
	}

	var dbgBI DebugBuildInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &dbgBI); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(dbgBI.BuildInfo) == 0 {
		t.Errorf("unexpected empty buildinfo")
	}
}
