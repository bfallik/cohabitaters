package main

import (
	"context"
	"encoding/base64"
	"io"
	"log"
	"os"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/cohabdb"
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

	cookieStoreKey, ok := os.LookupEnv("COOKIE_HASH_BLOCK_KEYS")
	if !ok {
		log.Fatalf("empty COOKIE_HASH_BLOCK_KEYS")
	}
	keys, err := base64.StdEncoding.DecodeString(cookieStoreKey)
	if err != nil {
		log.Fatalf("unable to decode COOKIE_HASH_BLOCK_KEYS")
	}
	if len(keys) != 96 {
		log.Fatalf("unexpected key length: %d", len(keys))
	}
	store := sessions.NewCookieStore(keys[0:64], keys[64:96])

	userCache := mapcache.Map[cohabitaters.UserState]{}

	db, err := cohabdb.Open()
	if err != nil {
		log.Fatalf("database open: %v", err)
	}
	ctx := context.Background()
	if err := cohabdb.CreateTables(ctx, db); err != nil {
		log.Fatalf("unable to create tables: %v", err)
	}
	queries := cohabdb.New(db)

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
		Queries:     queries,
	}

	webUIHandler := handlers.WebUI{
		OauthConfig: oauthConfig,
		UserCache:   &userCache,
		Queries:     queries,
	}

	e.GET("/static/fontawesome/*", handlers.FontAwesome)
	e.GET("/static/tailwindcss/*", handlers.Tailwind)

	e.GET("/", webUIHandler.Root)
	e.GET("/partial/tableResults", webUIHandler.PartialTableResults)
	e.GET("/about", webUIHandler.About)
	e.GET("/error", webUIHandler.Error)
	e.GET("/error2", webUIHandler.Error2)
	e.GET("/logout", webUIHandler.Logout)

	e.GET("/auth/google/callback", oauthHandler.GoogleCallbackAuthz).Name = handlers.RedirectURLAuthz
	e.GET("/auth/google/login", oauthHandler.GoogleLoginAuthz).Name = handlers.RedirectURLAuthzLogin
	e.GET("/auth/google/force-approval", oauthHandler.GoogleForceApproval)

	e.POST("/authn/google/callback", oauthHandler.GoogleCallbackAuthn).Name = handlers.RedirectURLAuthn

	e.GET("/debug/buildinfo", dbgHandler.BuildInfo)
	e.GET("/debug/sessions", dbgHandler.Sessions)

	e.Logger.Fatal(e.Start(listenAddress))
}
