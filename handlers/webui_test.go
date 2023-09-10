package handlers

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/cohabdb"
	"github.com/bfallik/cohabitaters/html"
	"github.com/bfallik/cohabitaters/mapcache"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
)

type renderFunc func(w io.Writer, name string, data interface{}, c echo.Context) error

func (f renderFunc) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return f(w, name, data, c)
}

func isValidHTML(r io.Reader) error {
	d := xml.NewDecoder(r)
	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity
	for {
		_, err := d.Token()
		switch err {
		case io.EOF:
			return nil
		case nil:
		default:
			return err
		}
	}
}

type mockSessioner struct{}

func (ms mockSessioner) ExpireSession(ctx context.Context, sessionID int64) error { return nil }
func (ms mockSessioner) GetSession(ctx context.Context, sessionID int64) (cohabdb.Session, error) {
	return cohabdb.Session{}, nil
}

func TestRoot(t *testing.T) {
	e := echo.New()
	e.Renderer = renderFunc(func(w io.Writer, name string, data interface{}, c echo.Context) error {
		return html.NewTemplater(html.Templates...).Render(w, name, data)
	})
	sess := mockSessioner{}

	subtester := func(cookie *http.Cookie) func(t *testing.T) {
		return func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/debug/buildinfo", nil)
			rec := httptest.NewRecorder()

			if cookie != nil {
				req.AddCookie(cookie)
			}

			c := e.NewContext(req, rec)

			store := sessions.NewCookieStore([]byte{})
			c.Set("_session_store", store)

			userCache := mapcache.Map[cohabitaters.UserState]{}
			h := &WebUI{
				UserCache: &userCache,
				Queries:   sess,
			}

			if err := h.Root(c); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if http.StatusOK != rec.Code {
				t.Errorf("expected:200, got: %v", rec.Code)
			}

			if err := isValidHTML(rec.Body); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			cookies := rec.Result().Cookies()
			if err := containsSessionCookie(cookies); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
	}

	cookie := new(http.Cookie)
	cookie.Name = sessionName

	t.Run("root is valid HTML", subtester(nil))
	t.Run("root handles invalid cookie", subtester(cookie))
}

func containsSessionCookie(cookies []*http.Cookie) error {
	if !slices.ContainsFunc(cookies, func(c *http.Cookie) bool { return c.Name == sessionName }) {
		return fmt.Errorf("unable to find '%s' in %v", sessionName, cookies)
	}
	return nil
}
