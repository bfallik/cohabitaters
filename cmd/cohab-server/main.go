package main

import (
	"encoding/base64"
	"io"
	"log"
	"os"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/handlers"
	"github.com/bfallik/cohabitaters/html"
	"github.com/bfallik/cohabitaters/mapcache"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/people/v1"
)

const defaultListenAddress = "localhost:8080"

type renderFunc func(w io.Writer, name string, data interface{}, c echo.Context) error

func (f renderFunc) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return f(w, name, data, c)
}

func main() {
	log.Printf("%s", cohabitaters.BuildInfo())

	listenAddress, ok := os.LookupEnv("LISTEN_ADDRESS")
	if !ok {
		listenAddress = defaultListenAddress
	}

	googleAppCredentials := os.Getenv("GOOGLE_APP_CREDENTIALS")
	oauthConfig, err := google.ConfigFromJSON([]byte(googleAppCredentials), people.ContactsReadonlyScope, people.UserinfoEmailScope)
	if err != nil {
		log.Fatalf("unable to create Google oauth2 config: %v", err)
	}
	oauthConfig.Endpoint = google.Endpoint

	cookieStoreKey, ok := os.LookupEnv("COOKIE_STORE_KEY")
	if !ok {
		log.Fatalf("empty COOKIE_STORE_KEY")
	}
	hashKey, err := base64.StdEncoding.DecodeString(cookieStoreKey)
	if err != nil {
		log.Fatalf("unable to decode COOKIE_STORE_KEY")
	}
	store := sessions.NewCookieStore(hashKey)

	userCache := mapcache.Map[cohabitaters.UserState]{}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(session.Middleware(store))
	e.Use(middleware.Secure())
	e.Renderer = renderFunc(func(w io.Writer, name string, data interface{}, c echo.Context) error {
		return html.NewTemplater(html.Templates...).Render(w, name, data)
	})

	dbgHandler := handlers.Debug{
		UserCache: &userCache,
	}

	oauthHandler := handlers.Oauth2{
		OauthConfig: oauthConfig,
		UserCache:   &userCache,
	}

	webUIHandler := handlers.WebUI{
		OauthConfig: oauthConfig,
		UserCache:   &userCache,
	}

	e.GET("/static/fontawesome/*", handlers.FontAwesome)

	e.GET("/", webUIHandler.Root)
	e.GET("/partial/tableResults", webUIHandler.PartialTableResults)
	e.GET("/about", webUIHandler.About)
	e.GET("/error", webUIHandler.Error)
	e.GET("/logout", webUIHandler.Logout)

	e.GET("/auth/google/callback", oauthHandler.GoogleCallback).Name = "redirectURL"
	e.GET("/auth/google/login", oauthHandler.NewGoogleLogin(e.Reverse("redirectURL")))
	e.GET("/auth/google/force-approval", oauthHandler.GoogleForceApproval)

	e.GET("/debug/buildinfo", dbgHandler.BuildInfo)
	e.GET("/debug/sessions", dbgHandler.Sessions)

	e.Logger.Fatal(e.Start(listenAddress))
}
