package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/html"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	oauth2_api "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const hostname = "localhost"
const port = "8080"

var googleOauthConfig *oauth2.Config

type renderBridge struct {
	*html.Templater
}

func (t renderBridge) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templater.Render(w, name, data)
}

const oauthCookieName = "oauthStateCookie"

func newStateAuthCookie() *http.Cookie {
	cookie := new(http.Cookie)
	cookie.Name = oauthCookieName
	cookie.Value = cohabitaters.RandBase64()
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.Path = "/"
	cookie.Domain = hostname
	cookie.Secure = true
	cookie.HttpOnly = true
	return cookie
}

func getUserDataFromGoogle(ctx context.Context, code string) (*oauth2_api.Userinfo, error) {
	token, err := googleOauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange error: %w", err)
	}

	tokenSource := googleOauthConfig.TokenSource(ctx, token)
	oauth2Service, err := oauth2_api.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	userInfoService := oauth2_api.NewUserinfoV2MeService(oauth2Service)
	return userInfoService.Get().Do()
}

func main() {

	creds, err := cohabitaters.ConfigFromJSONFile("client_secret.json")
	if err != nil {
		log.Fatal(err)
	}

	store := sessions.NewCookieStore([]byte(cohabitaters.RandBase64()))

	e := echo.New()
	e.Use(session.Middleware(store))
	e.Renderer = renderBridge{html.NewTemplater(html.Templates...)}

	faHandler := http.StripPrefix("/static/fontawesome/", http.FileServer(http.FS(html.FontAwesomeFS)))
	e.GET("/static/fontawesome/*", echo.WrapHandler(faHandler))

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
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
		u := googleOauthConfig.AuthCodeURL(oauthState.Value, oauth2.AccessTypeOnline)
		return c.Redirect(http.StatusTemporaryRedirect, u)
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

		s, _ := session.Get("default_session", c)
		s.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
		}
		s.Values["code"] = code
		if err := s.Save(c.Request(), c.Response()); err != nil {
			return err
		}

		return c.Redirect(http.StatusTemporaryRedirect, "/query")
	}).Name = "redirectURL"

	e.GET("/query", func(c echo.Context) error {
		s, _ := session.Get("default_session", c)
		codeVar, ok := s.Values["code"]
		if !ok {
			return fmt.Errorf("missing code from session")
		}

		code, ok := codeVar.(string)
		if !ok {
			return fmt.Errorf("unexpected type for code")
		}

		data, err := getUserDataFromGoogle(c.Request().Context(), code)
		if err != nil {
			return err
		}

		log.Println("Userinfo: ", data)
		return nil
	})

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
