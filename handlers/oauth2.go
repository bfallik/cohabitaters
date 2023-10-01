package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bfallik/cohabitaters/cohabdb"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const (
	oauthCookieName       = "oauthStateCookie"
	RedirectURLAuthn      = "redirectURLAuthn"
	RedirectURLAuthz      = "redirectURLAuthz"
	RedirectURLAuthzLogin = "redirectURLAuthzLogin"
)

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

func getContactGroupsList(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (*people.ListContactGroupsResponse, error) {
	tokenSource := cfg.TokenSource(ctx, token)
	srv, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("unable to create people service %w", err)
	}

	return srv.ContactGroups.List().Do()
}

func mustRandInt() int {
	n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		panic(fmt.Sprintf("unable to generate random int: %v", err))
	}
	return int(n.Int64())
}

// always returns a new or existing session ID
func sessionID(s *sessions.Session) int {
	idVar, ok := s.Values["id"]
	if !ok {
		return mustRandInt()
	}

	id, err := strconv.Atoi(idVar.(string))
	if err != nil {
		return mustRandInt()
	}
	return id
}

type Oauth2 struct {
	OauthConfig *oauth2.Config
	Queries     *cohabdb.Queries
}

func (o *Oauth2) GoogleLoginAuthz(c echo.Context) error {
	host := c.Request().Host

	oauthState := newStateAuthCookie(host)
	c.SetCookie(oauthState)

	callback := url.URL{
		Scheme: c.Request().Header.Get("X-Forwarded-Proto"),
		Host:   host,
		Path:   c.Echo().Reverse(RedirectURLAuthz),
	}
	if callback.Scheme == "" {
		callback.Scheme = "http"
	}
	o.OauthConfig.RedirectURL = callback.String()

	s, err := session.Get("default_session", c)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}

	sessionID := sessionID(s)
	session, err := o.Queries.GetSession(c.Request().Context(), int64(sessionID))
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}

	/*
		AuthCodeURL receive state that is a token to protect the user from CSRF attacks. You must always provide a non-empty string and
		validate that it matches the the state query parameter on your redirect callback.
	*/
	opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOnline}
	if session.GoogleForceApproval {
		opts = append(opts, oauth2.ApprovalForce)
	}
	u := o.OauthConfig.AuthCodeURL(oauthState.Value, opts...)
	return c.Redirect(http.StatusTemporaryRedirect, u)
}

func (o *Oauth2) GoogleForceApproval(c echo.Context) error {
	s, err := session.Get("default_session", c)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}
	sessionID := sessionID(s)
	session, err := o.Queries.GetSession(c.Request().Context(), int64(sessionID))
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}

	if err := o.Queries.UpdateGoogleForceApproval(c.Request().Context(), cohabdb.UpdateGoogleForceApprovalParams{
		ID:                  session.ID,
		GoogleForceApproval: !session.GoogleForceApproval,
	}); err != nil {
		return fmt.Errorf("error setting GoogleForceApproval: %w", err)
	}

	return c.JSON(http.StatusOK, struct{ ForceApproval bool }{!session.GoogleForceApproval})
}

func (o *Oauth2) setGoogleToken(ctx context.Context, sessionID int, tok *oauth2.Token) error {
	bs, err := json.Marshal(tok)
	if err != nil {
		return err
	}

	utp := cohabdb.UpdateTokenBySessionParams{
		ID:    int64(sessionID),
		Token: sql.NullString{String: string(bs), Valid: true},
	}
	if err := o.Queries.UpdateTokenBySession(ctx, utp); err != nil {
		return err
	}

	return nil
}

func (o *Oauth2) GoogleCallbackAuthz(c echo.Context) error {
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

	bs, err := json.Marshal(userGroups)
	if err != nil {
		return fmt.Errorf("error marshaling userGroups: %w", err)
	}

	if err := o.Queries.UpdateContactGroupsJSON(ctx, cohabdb.UpdateContactGroupsJSONParams{
		ID:                int64(sessionID),
		ContactGroupsJson: sql.NullString{Valid: true, String: string(bs)},
	}); err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}

	if err := o.setGoogleToken(ctx, sessionID, token); err != nil {
		return fmt.Errorf("error saving token: %w", err)
	}

	s.Values["id"] = fmt.Sprint(sessionID)
	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func mapGet[T any](m map[string]interface{}, key string) (T, bool) {
	var zero T
	v, ok := m[key]
	if !ok {
		return zero, false
	}
	ret, ok := v.(T)
	return ret, ok
}

func claimToNullString(m map[string]interface{}, key string) (result sql.NullString) {
	if name, ok := mapGet[string](m, key); ok { // found the key
		result = sql.NullString{String: name, Valid: true}
	}
	return
}

func unmarshalClaims(m map[string]interface{}, cup *cohabdb.CreateUserParams) error {
	sub, ok := mapGet[string](m, "sub")
	if !ok {
		return fmt.Errorf("sub claim not found")
	}
	cup.Sub = sub

	cup.Name = claimToNullString(m, "name")
	cup.Picture = claimToNullString(m, "picture")

	return nil
}

func (o *Oauth2) LogUserIn(ctx context.Context, cup cohabdb.CreateUserParams, sessionID int) (cohabdb.Session, error) {
	user, err := cohabdb.CreateOrSelectUser(ctx, o.Queries, cup)
	if err != nil {
		return cohabdb.Session{}, fmt.Errorf("error upserting user: %v", err)
	}

	return o.Queries.CreateSession(ctx, cohabdb.CreateSessionParams{
		ID: int64(sessionID),
		UserID: sql.NullInt64{
			Int64: user.ID,
			Valid: true,
		},
	})
}

func (o *Oauth2) GoogleCallbackAuthn(c echo.Context) error {
	csrfTokenCookie, err := c.Cookie("g_csrf_token")
	if err != nil {
		return fmt.Errorf("g_csrf_token cookie not found")
	}

	csrfTokenBody := c.FormValue("g_csrf_token")
	if len(csrfTokenBody) == 0 {
		return fmt.Errorf("g_csrf_token body not found")
	}

	if csrfTokenCookie.Value != csrfTokenBody {
		return fmt.Errorf("g_csrf_token mismatch")
	}

	credential := c.FormValue("credential")
	ctx := c.Request().Context()

	val, err := idtoken.NewValidator(ctx)
	if err != nil {
		return fmt.Errorf("error creating validator: %v", err)
	}

	pay, err := val.Validate(ctx, credential, clientID)
	if err != nil {
		return fmt.Errorf("error creating validator: %v", err)
	}

	var cup cohabdb.CreateUserParams
	if err := unmarshalClaims(pay.Claims, &cup); err != nil {
		return err
	}

	s, err := session.Get("default_session", c)
	if err != nil {
		return fmt.Errorf("error getting session: %w", err)
	}
	sessionID := sessionID(s)

	_, err = o.LogUserIn(ctx, cup, sessionID)
	if err != nil {
		return fmt.Errorf("error logging in: %v", err)
	}

	s.Values["id"] = fmt.Sprint(sessionID)
	if err := s.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, c.Echo().Reverse(RedirectURLAuthzLogin))
}
