package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"

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
	Queries     *cohabdb.Queries
}

func (w WebUI) getContacts(ctx context.Context, u cohabitaters.UserState, tmplData html.TmplIndexData) (html.TmplIndexData, error) {
	if u.Token != nil && u.Token.Valid() && len(u.SelectedResourceName) > 0 {
		idx := contactGroupIndex(u.ContactGroups, u.SelectedResourceName)
		cg := u.ContactGroups[idx]

		googs := googleSvcs{TokenSource: w.OauthConfig.TokenSource(ctx, u.Token)}
		cards, err := googs.getContacts(ctx, u.SelectedResourceName)
		if err != nil {
			if errors.Is(err, cohabitaters.ErrEmptyGroup) {
				tmplData.GroupErrorMsg = fmt.Sprintf("No contacts found in group <%s>", cg.Name)
				return tmplData, nil
			}
			return tmplData, err
		}
		tmplData.TableResults = cards
		tmplData.CountContacts = int(cg.MemberCount)
	}
	return tmplData, nil
}

func (w WebUI) Root(c echo.Context) error {
	s, err := session.Get(sessionName, c)
	if err != nil {
		c.Logger().Infof("error getting previous session: %w", err)
	}

	s.Options.HttpOnly = true

	sessionID := sessionID(s)
	userState := w.UserCache.Get(sessionID)

	u := new(url.URL)
	u.Host = c.Request().Host
	u.Path = c.Echo().Reverse(RedirectURLAuthn)

	tmplData := html.TmplIndexData{
		WelcomeMsg:           "Welcome",
		LoginURL:             u.String(),
		ClientID:             clientID,
		Groups:               userState.ContactGroups,
		SelectedResourceName: userState.SelectedResourceName,
	}

	if userState.Userinfo != nil {
		tmplData.WelcomeMsg = fmt.Sprintf("Welcome %s", userState.Userinfo.Email)
	}

	if tmplData, err = w.getContacts(c.Request().Context(), userState, tmplData); err != nil {
		return err
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

	userState.SelectedResourceName = c.QueryParam("contact-group")
	w.UserCache.Set(sessionID, userState)

	tmplData := html.TmplIndexData{
		Groups:               userState.ContactGroups,
		SelectedResourceName: userState.SelectedResourceName,
	}

	if tmplData, err = w.getContacts(c.Request().Context(), userState, tmplData); err != nil {
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

	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}
