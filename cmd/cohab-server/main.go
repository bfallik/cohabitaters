package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/handlers"
	"github.com/bfallik/cohabitaters/html"
	"github.com/bfallik/cohabitaters/mapcache"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const defaultListenAddress = "localhost:8080"

type renderFunc func(w io.Writer, name string, data interface{}, c echo.Context) error

func (f renderFunc) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return f(w, name, data, c)
}

func getContacts(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token, contactGroupResource string) ([]cohabitaters.XmasCard, error) {
	tokenSource := cfg.TokenSource(ctx, token)
	srv, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("unable to create people service %w", err)
	}

	return cohabitaters.GetXmasCards(srv, contactGroupResource)
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

	hashKey := securecookie.GenerateRandomKey(32)
	if hashKey == nil {
		log.Fatal("unable to generate initial random keys")
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

	e.GET("/static/fontawesome/*", handlers.FontAwesome)

	e.GET("/", func(c echo.Context) error {
		s, err := session.Get("default_session", c)
		if err != nil {
			return fmt.Errorf("error getting session: %w", err)
		}
		sessionID := sessionID(s)
		userState := userCache.Get(sessionID)

		tmplData := newTmplIndexData(userState)
		if tmplData, err = getContactsFromUserState(c.Request().Context(), userState, oauthConfig, tmplData); err != nil {
			return err
		}

		s.Values["id"] = fmt.Sprint(sessionID)
		if err := s.Save(c.Request(), c.Response()); err != nil {
			return err
		}
		return c.Render(http.StatusOK, "index.html", tmplData)
	})

	e.GET("/partial/tableResults", func(c echo.Context) error {
		s, err := session.Get("default_session", c)
		if err != nil {
			return fmt.Errorf("error getting session: %w", err)
		}
		sessionID := sessionID(s)
		userState := userCache.Get(sessionID)

		userState.SelectedResourceName = c.QueryParam("contact-group")
		userCache.Set(sessionID, userState)

		tmplData := newTmplIndexData(userState)
		if tmplData, err = getContactsFromUserState(c.Request().Context(), userState, oauthConfig, tmplData); err != nil {
			return err
		}

		return c.Render(http.StatusOK, "partials/results.html", tmplData)
	})

	e.GET("/about", func(c echo.Context) error {
		return c.Render(http.StatusOK, "about.html", nil)
	})

	e.GET("/error", func(c echo.Context) error {
		return c.Render(http.StatusInternalServerError, "error.html", nil)
	})

	e.GET("/logout", func(c echo.Context) error {
		s, err := session.Get("default_session", c)
		if err != nil {
			return fmt.Errorf("error getting session: %w", err)
		}

		s.Options.MaxAge = -1

		sessionID := sessionID(s)
		userCache.Delete(sessionID)

		if err := s.Save(c.Request(), c.Response()); err != nil {
			return err
		}

		return c.Redirect(http.StatusTemporaryRedirect, "/")
	})

	e.GET("/auth/google/callback", oauthHandler.GoogleCallback).Name = "redirectURL"
	e.GET("/auth/google/login", oauthHandler.NewGoogleLogin(e.Reverse("redirectURL")))
	e.GET("/auth/google/force-approval", oauthHandler.GoogleForceApproval)

	e.GET("/debug/buildinfo", dbgHandler.BuildInfo)
	e.GET("/debug/sessions", dbgHandler.Sessions)

	e.Logger.Fatal(e.Start(listenAddress))
}
