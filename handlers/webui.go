package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/html"
	"github.com/bfallik/cohabitaters/mapcache"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

func getContacts(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token, contactGroupResource string) ([]cohabitaters.XmasCard, error) {
	tokenSource := cfg.TokenSource(ctx, token)
	srv, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("unable to create people service %w", err)
	}

	return cohabitaters.GetXmasCards(srv, contactGroupResource)
}

func newTmplIndexData(u cohabitaters.UserState) html.TmplIndexData {
	res := html.TmplIndexData{
		WelcomeMsg:           "Welcome",
		Groups:               u.ContactGroups,
		SelectedResourceName: u.SelectedResourceName,
	}

	if u.Userinfo != nil {
		res.WelcomeMsg = fmt.Sprintf("Welcome %s", u.Userinfo.Email)
	}

	return res
}

func lookupContactGroup(cgs []*people.ContactGroup, resName string) *people.ContactGroup {
	for _, cg := range cgs {
		if cg.ResourceName == resName {
			return cg
		}
	}
	panic("resource name not found")
}

func getContactsFromUserState(ctx context.Context, u cohabitaters.UserState, cfg *oauth2.Config, tmplData html.TmplIndexData) (html.TmplIndexData, error) {
	if u.Token != nil && u.Token.Valid() && len(u.SelectedResourceName) > 0 {
		cg := lookupContactGroup(u.ContactGroups, u.SelectedResourceName)

		cards, err := getContacts(ctx, cfg, u.Token, u.SelectedResourceName)
		if err != nil {
			if errors.Is(err, cohabitaters.ErrEmptyGroup) {
				tmplData.GroupErrorMsg = fmt.Sprintf("No contacts found in group <%s>", cg.Name)
				return tmplData, nil
			}
			return tmplData, err
		}
		tmplData.TableResults = cards
		tmplData.SelectedResourceName = u.SelectedResourceName
		tmplData.CountContacts = int(cg.MemberCount)
		tmplData.CountAddresses = len(cards)
	}
	return tmplData, nil
}

type WebUI struct {
	OauthConfig *oauth2.Config
	UserCache   *mapcache.Map[cohabitaters.UserState]
}

func (w WebUI) Root(c echo.Context) error {
	s, err := session.Get("default_session", c)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}
	sessionID := sessionID(s)
	userState := w.UserCache.Get(sessionID)

	tmplData := newTmplIndexData(userState)
	if tmplData, err = getContactsFromUserState(c.Request().Context(), userState, w.OauthConfig, tmplData); err != nil {
		return err
	}

	s.Values["id"] = fmt.Sprint(sessionID)
	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}
	return c.Render(http.StatusOK, "index.html", tmplData)
}

func (w WebUI) PartialTableResults(c echo.Context) error {

	s, err := session.Get("default_session", c)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}
	sessionID := sessionID(s)
	userState := w.UserCache.Get(sessionID)

	userState.SelectedResourceName = c.QueryParam("contact-group")
	w.UserCache.Set(sessionID, userState)

	tmplData := newTmplIndexData(userState)
	if tmplData, err = getContactsFromUserState(c.Request().Context(), userState, w.OauthConfig, tmplData); err != nil {
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
	s, err := session.Get("default_session", c)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}

	s.Options.MaxAge = -1

	sessionID := sessionID(s)
	w.UserCache.Delete(sessionID)

	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}