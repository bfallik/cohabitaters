package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/mapcache"
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

func TestDebugSessions(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/debug/sessions", nil)
	userCache := mapcache.Map[cohabitaters.UserState]{}
	h := &Debug{
		UserCache: &userCache,
	}
	var dbgSess DebugSessions

	assertJSON := func(t *testing.T, rec *httptest.ResponseRecorder, target interface{}) {
		t.Helper()
		if http.StatusOK != rec.Code {
			t.Errorf("expected:200, got: %v", rec.Code)
		}

		if err := json.Unmarshal(rec.Body.Bytes(), &dbgSess); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}

	t.Run("empty sessions", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := h.Sessions(c); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		assertJSON(t, rec, dbgSess)

		if len(dbgSess.Sessions) != 0 {
			t.Errorf("unexpected empty session, got %v:", dbgSess.Sessions)
		}
	})

	t.Run("non-empty sessions", func(t *testing.T) {
		userCache.Set(11, cohabitaters.UserState{})
		userCache.Set(12, cohabitaters.UserState{})
		userCache.Set(13, cohabitaters.UserState{})

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := h.Sessions(c); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		assertJSON(t, rec, dbgSess)

		if len(dbgSess.Sessions) != 3 {
			t.Errorf("expected: 3, got: %v:", dbgSess.Sessions)
		}

	})
}
