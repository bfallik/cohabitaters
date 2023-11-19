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

	"github.com/a-h/templ"
	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/cohabdb"
	"github.com/bfallik/cohabitaters/html"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const sessionName = "default_session"
const clientID = "1048297799487-pibn8vimfmlii915gn5frkjgorq3oqhn.apps.googleusercontent.com"
const sessionTimeout = 600 * time.Second

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
	Queries     cohabdb.Querier
}

func newTmplIndexData() html.TmplIndexData {
	return html.TmplIndexData{
		ClientID: clientID,
	}
}

func (w WebUI) fillTmplIndexData(ctx context.Context, sessionID int, selectedResourceName string, out *html.TmplIndexData) error {
	token, err := w.getGoogleToken(ctx, sessionID)
	if err != nil {
		return err
	}

	session, err := w.Queries.GetSession(ctx, int64(sessionID))
	if err != nil {
		return err
	}

	var groups []*people.ContactGroup
	if session.ContactGroupsJson.Valid {
		if err := json.Unmarshal([]byte(session.ContactGroupsJson.String), &groups); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid ContactGroups JSON")
	}
	out.Groups = groups

	if len(groups) == 0 {
		return nil
	}

	if len(selectedResourceName) == 0 && session.SelectedResourceName.Valid {
		selectedResourceName = session.SelectedResourceName.String
	}

	if token.Valid() && len(selectedResourceName) > 0 {
		idx := contactGroupIndex(groups, selectedResourceName)
		cg := groups[idx]

		googs := googleSvcs{TokenSource: w.OauthConfig.TokenSource(ctx, token)}
		cards, err := googs.getContacts(ctx, selectedResourceName)
		if err != nil {
			if errors.Is(err, cohabitaters.ErrEmptyGroup) {
				out.GroupErrorMsg = fmt.Sprintf("No contacts found in group <%s>", cg.Name)
				return nil
			}
			return err
		}
		out.TableResults = cards
		out.CountContacts = int(cg.MemberCount)
		out.SelectedResourceName = selectedResourceName
	}

	return nil
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

func (w WebUI) getUserName(ctx context.Context, sessionID int) (sql.NullString, error) {
	user, err := w.Queries.GetUserBySession(ctx, int64(sessionID))
	if err != nil {
		return sql.NullString{}, err
	}

	if !user.Name.Valid {
		return sql.NullString{}, err
	}

	return user.Name, err
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

func renderComponentHTML(c echo.Context, cmp templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	return cmp.Render(c.Request().Context(), c.Response().Writer)
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

	tmplData := newTmplIndexData()
	u := new(url.URL)
	u.Host = c.Request().Host
	u.Path = c.Echo().Reverse(RedirectURLAuthn)
	tmplData.LoginURL = u.String()
	tmplData.IsLoggedIn = isLoggedIn

	if isLoggedIn {
		if err = w.fillTmplIndexData(c.Request().Context(), sessionID, "", &tmplData); err != nil {
			return err
		}

		name, err := w.getUserName(ctx, sessionID)
		if err != nil {
			return err
		}

		if name.Valid {
			tmplData.WelcomeName = name.String
		}
	}

	s.Values["id"] = fmt.Sprint(sessionID)
	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	return renderComponentHTML(c, html.ComponentPageIndex(tmplData))
}

func (w WebUI) PartialTableResults(c echo.Context) error {

	s, err := session.Get(sessionName, c)
	if err != nil {
		c.Logger().Infof("error getting previous session: %w", err)
	}
	sessionID := sessionID(s)

	ctx := c.Request().Context()
	isLoggedIn, err := w.isUserLoggedIn(ctx, sessionID)
	if err != nil {
		return err
	}
	if !isLoggedIn {
		c.Logger().Infof("request for partial results without login session")
		return c.Render(http.StatusUnauthorized, "error.html", nil)
	}

	selectedResourceName, ok := c.QueryParams()["contact-group"]
	if !ok {
		c.Logger().Error("missing expected contact-group")
		return c.NoContent(http.StatusBadRequest)
	}

	tmplData := newTmplIndexData()
	if err = w.fillTmplIndexData(c.Request().Context(), sessionID, selectedResourceName[0], &tmplData); err != nil {
		return err
	}

	if err := w.Queries.UpdateSelectedResourceName(
		ctx,
		cohabdb.UpdateSelectedResourceNameParams{
			ID:                   int64(sessionID),
			SelectedResourceName: sql.NullString{Valid: true, String: selectedResourceName[0]}}); err != nil {
		return err
	}

	return renderComponentHTML(c, html.ComponentTableResults(tmplData))
}

func (w WebUI) Logout(c echo.Context) error {
	s, err := session.Get(sessionName, c)
	if err != nil {
		c.Logger().Infof("error getting previous session: %w", err)
	}

	s.Options.MaxAge = -1

	sessionID := sessionID(s)

	if err := w.logUserOut(c.Request().Context(), sessionID); err != nil {
		return err
	}

	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}
