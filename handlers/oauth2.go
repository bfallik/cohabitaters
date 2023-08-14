package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/mapcache"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	oauth2_api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const oauthCookieName = "oauthStateCookie"

func newStateAuthCookie(domain string) *http.Cookie {
	bs := securecookie.GenerateRandomKey(32)
	if bs == nil {
		panic("unable to allocate random bytes")
	}

	cookie := new(http.Cookie)
	cookie.Name = oauthCookieName
	cookie.Value = base64.URLEncoding.EncodeToString(bs)
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.Path = "/"
	cookie.Domain = domain
	cookie.Secure = true
	cookie.HttpOnly = true
	return cookie
}

func getUserinfo(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (*oauth2_api.Userinfo, error) {
	tokenSource := cfg.TokenSource(ctx, token)
	oauth2Service, err := oauth2_api.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	userInfoService := oauth2_api.NewUserinfoV2MeService(oauth2Service)
	return userInfoService.Get().Do()
}

func getContactGroupsList(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (*people.ListContactGroupsResponse, error) {
	tokenSource := cfg.TokenSource(ctx, token)
	srv, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("unable to create people service %w", err)
	}

	return srv.ContactGroups.List().Do()
}

// always returns a new or existing session ID
func sessionID(s *sessions.Session) int {
	idVar, ok := s.Values["id"]
	if !ok {
		return rand.Int()
	}

	id, err := strconv.Atoi(idVar.(string))
	if err != nil {
		return rand.Int()
	}
	return id
}

type Oauth2 struct {
	OauthConfig *oauth2.Config
	UserCache   *mapcache.Map[cohabitaters.UserState]
}

func (o *Oauth2) NewGoogleLogin(redirectURL string) echo.HandlerFunc {
	return func(c echo.Context) error {
		host := c.Request().Host

		oauthState := newStateAuthCookie(host)
		c.SetCookie(oauthState)

		localConfig := o.OauthConfig
		callback := url.URL{
			Scheme: c.Request().Header.Get("X-Forwarded-Proto"),
			Host:   host,
			Path:   redirectURL,
		}
		if callback.Scheme == "" {
			callback.Scheme = "http"
		}
		localConfig.RedirectURL = callback.String()

		/*
			AuthCodeURL receive state that is a token to protect the user from CSRF attacks. You must always provide a non-empty string and
			validate that it matches the the state query parameter on your redirect callback.
		*/
		s, err := session.Get("default_session", c)
		if err != nil {
			return fmt.Errorf("error getting session: %w", err)
		}
		sessionID := sessionID(s)
		userState := o.UserCache.Get(sessionID)

		var u string
		if userState.GoogleForceApproval {
			u = localConfig.AuthCodeURL(oauthState.Value, oauth2.AccessTypeOnline, oauth2.ApprovalForce)
		} else {
			u = localConfig.AuthCodeURL(oauthState.Value, oauth2.AccessTypeOnline)
		}
		return c.Redirect(http.StatusTemporaryRedirect, u)
	}
}

func (o *Oauth2) GoogleForceApproval(c echo.Context) error {
	s, err := session.Get("default_session", c)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}
	sessionID := sessionID(s)
	userState := o.UserCache.Get(sessionID)

	userState.GoogleForceApproval = !userState.GoogleForceApproval
	o.UserCache.Set(sessionID, userState)

	return c.JSON(http.StatusOK, struct{ ForceApproval bool }{userState.GoogleForceApproval})
}

func (o *Oauth2) GoogleCallback(c echo.Context) error {
	maybeError := c.QueryParam("error")
	if len(maybeError) > 0 {
		return fmt.Errorf("authorization error: %s", maybeError)
	}

	oauthState, err := c.Cookie(oauthCookieName)
	if err != nil {
		return fmt.Errorf("unable to retrieve %s cookie: %w", oauthCookieName, err)
	}

	if c.QueryParam("state") != oauthState.Value {
		return fmt.Errorf("mismatched oauth google state: %s != %s", c.QueryParam("state"), oauthState.Value)
	}
	oauthState.MaxAge = -1
	c.SetCookie(oauthState)

	code := c.QueryParam("code")
	if len(code) == 0 {
		return fmt.Errorf("empty code parameter")
	}

	ctx := c.Request().Context()
	token, err := o.OauthConfig.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("code exchange error: %w", err)
	}

	userinfo, err := getUserinfo(ctx, o.OauthConfig, token)
	if err != nil {
		return err
	}

	groupsResponse, err := getContactGroupsList(ctx, o.OauthConfig, token)
	if err != nil {
		return err
	}

	userGroups := []*people.ContactGroup{}
	for _, cg := range groupsResponse.ContactGroups {
		if cg.GroupType == "USER_CONTACT_GROUP" {
			userGroups = append(userGroups, cg)
		}
	}

	s, err := session.Get("default_session", c)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}
	sessionID := sessionID(s)
	userState := o.UserCache.Get(sessionID)

	userState.Token = token
	userState.Userinfo = userinfo
	userState.ContactGroups = userGroups
	o.UserCache.Set(sessionID, userState)

	s.Values["id"] = fmt.Sprint(sessionID)
	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}
