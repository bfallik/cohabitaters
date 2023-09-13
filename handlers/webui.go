package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/cohabdb"
	"github.com/bfallik/cohabitaters/html"
	"github.com/bfallik/cohabitaters/mapcache"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const sessionName = "default_session"
const clientID = "1048297799487-pibn8vimfmlii915gn5frkjgorq3oqhn.apps.googleusercontent.com"
const sessionTimeout = 60 * time.Second

type googleSvcs struct {
	TokenSource oauth2.TokenSource
}

func (gs googleSvcs) getContacts(ctx context.Context, contactGroupResource string) ([]cohabitaters.XmasCard, error) {
	srv, err := people.NewService(ctx, option.WithTokenSource(gs.TokenSource))
	if err != nil {
		return nil, fmt.Errorf("unable to create people service %w", err)
	}

	return cohabitaters.GetXmasCards(srv, contactGroupResource)
}

func contactGroupIndex(cgs []*people.ContactGroup, target string) int {
	return slices.IndexFunc(cgs, func(cg *people.ContactGroup) bool { return cg.ResourceName == target })
}

type WebUI struct {
	OauthConfig *oauth2.Config
	UserCache   *mapcache.Map[cohabitaters.UserState]
	Queries     queryer
}

func (w WebUI) newTmplIndexData(ctx context.Context, u cohabitaters.UserState, tok *oauth2.Token) (html.TmplIndexData, error) {
	result := html.TmplIndexData{
		ClientID:             clientID,
		Groups:               u.ContactGroups,
		SelectedResourceName: u.SelectedResourceName,
	}

	if tok.Valid() && len(u.SelectedResourceName) > 0 {
		idx := contactGroupIndex(u.ContactGroups, u.SelectedResourceName)
		cg := u.ContactGroups[idx]

		googs := googleSvcs{TokenSource: w.OauthConfig.TokenSource(ctx, tok)}
		cards, err := googs.getContacts(ctx, u.SelectedResourceName)
		if err != nil {
			if errors.Is(err, cohabitaters.ErrEmptyGroup) {
				result.GroupErrorMsg = fmt.Sprintf("No contacts found in group <%s>", cg.Name)
				return result, nil
			}
			return result, err
		}
		result.TableResults = cards
		result.CountContacts = int(cg.MemberCount)
	}

	return result, nil
}

type queryer interface {
	ExpireSession(ctx context.Context, sessionID int64) error
	GetSession(ctx context.Context, sessionID int64) (cohabdb.Session, error)
	GetToken(ctx context.Context, sessionID int64) (sql.NullString, error)
}

func (w WebUI) logUserOut(ctx context.Context, sessionID int) error {
	if err := w.Queries.ExpireSession(ctx, int64(sessionID)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return nil
}

func (w WebUI) isUserLoggedIn(ctx context.Context, sessionID int) (bool, error) {
	session, err := w.Queries.GetSession(ctx, int64(sessionID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}

	if !session.IsLoggedIn {
		return false, nil
	}

	return time.Since(time.Unix(session.CreatedAt, 0)) <= sessionTimeout, nil
}
func (w WebUI) getGoogleToken(ctx context.Context, sessionID int) (*oauth2.Token, error) {
	tokenText, err := w.Queries.GetToken(ctx, int64(sessionID))
	if err != nil {
		return nil, err
	}

	tok := oauth2.Token{}
	err = json.Unmarshal([]byte(tokenText.String), &tok)
	return &tok, err
}

func (w WebUI) Root(c echo.Context) error {
	s, err := session.Get(sessionName, c)
	if err != nil {
		c.Logger().Infof("error getting previous session: %w", err)
	}

	s.Options.HttpOnly = true

	sessionID := sessionID(s)

	ctx := c.Request().Context()
	isLoggedIn, err := w.isUserLoggedIn(ctx, sessionID)
	if err != nil {
		return err
	}

	var tmplData html.TmplIndexData
	u := new(url.URL)
	u.Host = c.Request().Host
	u.Path = c.Echo().Reverse(RedirectURLAuthn)
	tmplData.LoginURL = u.String()

	if isLoggedIn {
		userState := w.UserCache.Get(sessionID)
		token, err := w.getGoogleToken(ctx, sessionID)
		if err != nil {
			return err
		}
		if tmplData, err = w.newTmplIndexData(c.Request().Context(), userState, token); err != nil {
			return err
		}

		if userState.Userinfo != nil {
			tmplData.WelcomeName = userState.Userinfo.Email
		}
	}

	s.Values["id"] = fmt.Sprint(sessionID)
	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}
	return c.Render(http.StatusOK, "index.html", tmplData)
}

func (w WebUI) PartialTableResults(c echo.Context) error {

	s, err := session.Get(sessionName, c)
	if err != nil {
		c.Logger().Infof("error getting previous session: %w", err)
	}
	sessionID := sessionID(s)
	userState := w.UserCache.Get(sessionID)

	ctx := c.Request().Context()
	isLoggedIn, err := w.isUserLoggedIn(ctx, sessionID)
	if err != nil {
		return err
	}
	if !isLoggedIn {
		c.Logger().Infof("request for partial results without login session")
		return c.Render(http.StatusUnauthorized, "error.html", nil)
	}
	userState.SelectedResourceName = c.QueryParam("contact-group")
	w.UserCache.Set(sessionID, userState)

	token, err := w.getGoogleToken(ctx, sessionID)
	if err != nil {
		return err
	}

	var tmplData html.TmplIndexData
	if tmplData, err = w.newTmplIndexData(c.Request().Context(), userState, token); err != nil {
		return err
	}

	return c.Render(http.StatusOK, "partials/results.html", tmplData)
}

func (w WebUI) About(c echo.Context) error {
	return c.Render(http.StatusOK, "about.html", nil)
}

func (w WebUI) Error(c echo.Context) error {
	return c.Render(http.StatusInternalServerError, "error.html", nil)
}

func (w WebUI) Logout(c echo.Context) error {
	s, err := session.Get(sessionName, c)
	if err != nil {
		c.Logger().Infof("error getting previous session: %w", err)
	}

	s.Options.MaxAge = -1

	sessionID := sessionID(s)
	w.UserCache.Delete(sessionID)

	if err := w.logUserOut(c.Request().Context(), sessionID); err != nil {
		return err
	}

	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}
