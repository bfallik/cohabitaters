package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/html"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	oauth2_api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const hostname = "localhost"
const port = "8080"

var googleOauthConfig *oauth2.Config
var googleForceApproval bool // TODO: global for now

type renderBridge struct {
	*html.Templater
}

func (t renderBridge) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templater.Render(w, name, data)
}

const oauthCookieName = "oauthStateCookie"

func newStateAuthCookie() *http.Cookie {
	bs := securecookie.GenerateRandomKey(32)
	if bs == nil {
		panic("unable to allocated random bytes")
	}

	cookie := new(http.Cookie)
	cookie.Name = oauthCookieName
	cookie.Value = base64.URLEncoding.EncodeToString(bs)
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.Path = "/"
	cookie.Domain = hostname
	cookie.Secure = true
	cookie.HttpOnly = true
	return cookie
}

func getUserDataFromGoogle(ctx context.Context, tokenSource oauth2.TokenSource) (*oauth2_api.Userinfo, error) {
	oauth2Service, err := oauth2_api.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	userInfoService := oauth2_api.NewUserinfoV2MeService(oauth2Service)
	return userInfoService.Get().Do()
}

func getContactGroups(ctx context.Context, tokenSource oauth2.TokenSource) (*people.ListContactGroupsResponse, error) {
	srv, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		log.Fatalf("Unable to create people Client %v", err)
	}

	return srv.ContactGroups.List().Do()
}

func main() {

	creds, err := cohabitaters.ConfigFromJSONFile("client_secret.json")
	if err != nil {
		log.Fatal(err)
	}

	hashKey, blockKey := securecookie.GenerateRandomKey(32), securecookie.GenerateRandomKey(32)
	if hashKey == nil || blockKey == nil {
		log.Fatal("unable to generate initial random keys")
	}
	store := sessions.NewCookieStore(hashKey, blockKey)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(session.Middleware(store))
	e.Renderer = renderBridge{html.NewTemplater(html.Templates...)}

	faHandler := http.StripPrefix("/static/fontawesome/", http.FileServer(http.FS(html.FontAwesomeFS)))
	e.GET("/static/fontawesome/*", echo.WrapHandler(faHandler))

	e.GET("/", func(c echo.Context) error {
		tmplData := struct {
			WelcomeMsg string
			Groups     []*people.ContactGroup
		}{"Welcome", nil}

		s, _ := session.Get("default_session", c)
		tokenVar, ok := s.Values["token"]
		if !ok {
			return c.Render(http.StatusOK, "index.html", tmplData)
		}

		tokenJSON, ok := tokenVar.(string)
		if !ok {
			return fmt.Errorf("unexpected token type")
		}

		token := new(oauth2.Token)
		if err := json.Unmarshal([]byte(tokenJSON), token); err != nil {
			return fmt.Errorf("json unmarshal: %w", err)
		}

		ctx := c.Request().Context()
		tokenSource := googleOauthConfig.TokenSource(ctx, token)

		data, err := getUserDataFromGoogle(ctx, tokenSource)
		if err != nil {
			return err
		}

		groups, err := getContactGroups(ctx, tokenSource)
		if err != nil {
			return err
		}

		tmplData.WelcomeMsg = fmt.Sprintf("Welcome %s", data.Email)
		tmplData.Groups = groups.ContactGroups
		return c.Render(http.StatusOK, "index.html", tmplData)
	})

	e.GET("/error", func(c echo.Context) error {
		return c.Render(http.StatusInternalServerError, "error.html", nil)
	})

	e.GET("/auth/google/login", func(c echo.Context) error {
		oauthState := newStateAuthCookie()
		c.SetCookie(oauthState)

		/*
			AuthCodeURL receive state that is a token to protect the user from CSRF attacks. You must always provide a non-empty string and
			validate that it matches the the state query parameter on your redirect callback.
		*/
		var u string
		if googleForceApproval {
			u = googleOauthConfig.AuthCodeURL(oauthState.Value, oauth2.AccessTypeOnline, oauth2.ApprovalForce)
		} else {
			u = googleOauthConfig.AuthCodeURL(oauthState.Value, oauth2.AccessTypeOnline)
		}
		return c.Redirect(http.StatusTemporaryRedirect, u)
	})

	e.GET("/auth/google/force-approval", func(c echo.Context) error {
		googleForceApproval = !googleForceApproval
		return c.JSON(http.StatusOK, struct{ ForceApproval bool }{googleForceApproval})
	})

	e.GET("/auth/google/callback", func(c echo.Context) error {
		oauthState, err := c.Cookie(oauthCookieName)
		if err != nil {
			return fmt.Errorf("unable to retrieve %s cookie: %w", oauthCookieName, err)
		}

		if c.QueryParam("state") != oauthState.Value {
			return fmt.Errorf("mismatched oauth google state: %s != %s", c.QueryParam("state"), oauthState.Value)
		}

		code := c.QueryParam("code")
		if len(code) == 0 {
			return fmt.Errorf("empty code parameter")
		}

		ctx := c.Request().Context()
		token, err := googleOauthConfig.Exchange(ctx, code)
		if err != nil {
			return fmt.Errorf("code exchange error: %w", err)
		}

		tokenJSON, err := json.Marshal(token)
		if err != nil {
			return fmt.Errorf("json error: %w", err)
		}

		s, _ := session.Get("default_session", c)
		s.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
		}
		s.Values["token"] = string(tokenJSON)
		if err := s.Save(c.Request(), c.Response()); err != nil {
			return err
		}

		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}).Name = "redirectURL"

	serverAddress := net.JoinHostPort(hostname, port)
	redirectURL := url.URL{Scheme: "http", Host: serverAddress, Path: e.Reverse("redirectURL")}
	googleOauthConfig = &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Scopes:       []string{people.ContactsReadonlyScope, people.UserinfoEmailScope},
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL.String(),
	}

	e.Logger.Fatal(e.Start(serverAddress))
}
