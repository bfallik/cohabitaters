package main

import (
	"context"
	"encoding/base64"
	"log"
	"os"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/cohabdb"
	"github.com/bfallik/cohabitaters/handlers"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/people/v1"
)

const defaultListenAddress = "localhost:8080"
const defaultDBFile = "file:cohab.db"

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

	db, err := cohabdb.Open(defaultDBFile)
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

	dbgHandler := handlers.Debug{}

	oauthHandler := handlers.Oauth2{
		OauthConfig: oauthConfig,
		Queries:     queries,
	}

	webUIHandler := handlers.WebUI{
		OauthConfig: oauthConfig,
		Queries:     queries,
	}

	e.GET("/static/fontawesome/*", handlers.FontAwesome)
	e.GET("/static/tailwindcss/*", handlers.Tailwind)

	e.GET("/", webUIHandler.Root)
	e.GET("/partial/tableResults", webUIHandler.PartialTableResults)
	e.GET("/about", handlers.About)
	e.GET("/error", handlers.Error)
	e.GET("/logout", webUIHandler.Logout)

	e.GET("/auth/google/callback", oauthHandler.GoogleCallbackAuthz).Name = handlers.RedirectURLAuthz
	e.GET("/auth/google/login", oauthHandler.GoogleLoginAuthz).Name = handlers.RedirectURLAuthzLogin
	e.GET("/auth/google/force-approval", oauthHandler.GoogleForceApproval)

	e.POST("/authn/google/callback", oauthHandler.GoogleCallbackAuthn).Name = handlers.RedirectURLAuthn

	e.GET("/debug/buildinfo", dbgHandler.BuildInfo)

	e.Logger.Fatal(e.Start(listenAddress))
}
