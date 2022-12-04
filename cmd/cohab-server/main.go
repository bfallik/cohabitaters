package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/bfallik/cohabitaters"
	"github.com/bfallik/cohabitaters/html"
	"github.com/bfallik/cohabitaters/mapcache"
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

const defaultListenAddress = "localhost:8080"

var googleOauthConfig *oauth2.Config

type renderBridge struct {
	*html.Templater
}

func (t renderBridge) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templater.Render(w, name, data)
}

const oauthCookieName = "oauthStateCookie"

func newStateAuthCookie(domain string) *http.Cookie {
	bs := securecookie.GenerateRandomKey(32)
	if bs == nil {
		panic("unable to allocated random bytes")
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

func getUserinfo(ctx context.Context, token *oauth2.Token) (*oauth2_api.Userinfo, error) {
	tokenSource := googleOauthConfig.TokenSource(ctx, token)
	oauth2Service, err := oauth2_api.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, err
	}

	userInfoService := oauth2_api.NewUserinfoV2MeService(oauth2Service)
	return userInfoService.Get().Do()
}

func getContactGroupsList(ctx context.Context, token *oauth2.Token) (*people.ListContactGroupsResponse, error) {
	tokenSource := googleOauthConfig.TokenSource(ctx, token)
	srv, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("unable to create people service %w", err)
	}

	return srv.ContactGroups.List().Do()
}

func getContacts(ctx context.Context, token *oauth2.Token, contactGroupResource string) ([]cohabitaters.XmasCard, error) {
	tokenSource := googleOauthConfig.TokenSource(ctx, token)
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

type UserState struct {
	GoogleForceApproval  bool
	Token                *oauth2.Token
	Userinfo             *oauth2_api.Userinfo
	ContactGroups        []*people.ContactGroup
	SelectedResourceName string
}

func newTmplIndexData(u UserState) html.TmplIndexData {
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

func (u UserState) getContacts(ctx context.Context, tmplData html.TmplIndexData) (html.TmplIndexData, error) {
	if u.Token != nil && u.Token.Valid() && len(u.SelectedResourceName) > 0 {
		cards, err := getContacts(ctx, u.Token, u.SelectedResourceName)
		if err != nil {
			return tmplData, err
		}
		tmplData.TableResults = cards
		tmplData.SelectedResourceName = u.SelectedResourceName
	}
	return tmplData, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	listenAddress, ok := os.LookupEnv("LISTEN_ADDRESS")
	if !ok {
		listenAddress = defaultListenAddress
	}
	hostname, _, err := net.SplitHostPort(listenAddress)
	if err != nil {
		log.Fatalf("invalid listen address: %v", err)
	}

	googleAppCredentials := os.Getenv("GOOGLE_APP_CREDENTIALS")
	creds, err := google.ConfigFromJSON([]byte(googleAppCredentials))
	if err != nil {
		log.Fatalf("unable to create Google oauth2 config: %v", err)
	}

	hashKey := securecookie.GenerateRandomKey(32)
	if hashKey == nil {
		log.Fatal("unable to generate initial random keys")
	}
	store := sessions.NewCookieStore(hashKey)

	userCache := mapcache.Map[UserState]{}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(session.Middleware(store))
	e.Renderer = renderBridge{html.NewTemplater(html.Templates...)}

	faHandler := http.StripPrefix("/static/fontawesome/", http.FileServer(http.FS(html.FontAwesomeFS)))
	e.GET("/static/fontawesome/*", echo.WrapHandler(faHandler))

	e.GET("/", func(c echo.Context) error {
		s, _ := session.Get("default_session", c)
		sessionID := sessionID(s)
		userState := userCache.Get(sessionID)

		tmplData := newTmplIndexData(userState)
		if tmplData, err = userState.getContacts(c.Request().Context(), tmplData); err != nil {
			return err
		}

		s.Values["id"] = fmt.Sprint(sessionID)
		if err := s.Save(c.Request(), c.Response()); err != nil {
			return err
		}
		return c.Render(http.StatusOK, "index.html", tmplData)
	})

	e.GET("/partial/tableResults", func(c echo.Context) error {
		s, _ := session.Get("default_session", c)
		sessionID := sessionID(s)
		userState := userCache.Get(sessionID)

		userState.SelectedResourceName = c.QueryParam("contact-group")
		userCache.Set(sessionID, userState)

		tmplData := newTmplIndexData(userState)
		if tmplData, err = userState.getContacts(c.Request().Context(), tmplData); err != nil {
			return err
		}

		return c.Render(http.StatusOK, "partials/results.html", tmplData)
	})

	e.GET("/error", func(c echo.Context) error {
		return c.Render(http.StatusInternalServerError, "error.html", nil)
	})

	e.GET("/auth/google/login", func(c echo.Context) error {
		oauthState := newStateAuthCookie(hostname)
		c.SetCookie(oauthState)

		/*
			AuthCodeURL receive state that is a token to protect the user from CSRF attacks. You must always provide a non-empty string and
			validate that it matches the the state query parameter on your redirect callback.
		*/
		s, _ := session.Get("default_session", c)
		sessionID := sessionID(s)

		userState := userCache.Get(sessionID)

		var u string
		if userState.GoogleForceApproval {
			u = googleOauthConfig.AuthCodeURL(oauthState.Value, oauth2.AccessTypeOnline, oauth2.ApprovalForce)
		} else {
			u = googleOauthConfig.AuthCodeURL(oauthState.Value, oauth2.AccessTypeOnline)
		}
		return c.Redirect(http.StatusTemporaryRedirect, u)
	})

	e.GET("/auth/google/force-approval", func(c echo.Context) error {
		s, _ := session.Get("default_session", c)
		sessionID := sessionID(s)

		userState := userCache.Get(sessionID)

		userState.GoogleForceApproval = !userState.GoogleForceApproval
		userCache.Set(sessionID, userState)

		return c.JSON(http.StatusOK, struct{ ForceApproval bool }{userState.GoogleForceApproval})
	})

	e.GET("/auth/google/callback", func(c echo.Context) error {
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

		code := c.QueryParam("code")
		if len(code) == 0 {
			return fmt.Errorf("empty code parameter")
		}

		ctx := c.Request().Context()
		token, err := googleOauthConfig.Exchange(ctx, code)
		if err != nil {
			return fmt.Errorf("code exchange error: %w", err)
		}

		userinfo, err := getUserinfo(ctx, token)
		if err != nil {
			return err
		}

		groupsResponse, err := getContactGroupsList(ctx, token)
		if err != nil {
			return err
		}

		s, err := session.Get("default_session", c)
		if err != nil {
			return fmt.Errorf("error getting session: %w", err)
		}
		sessionID := sessionID(s)

		userState := userCache.Get(sessionID)
		userState.Token = token
		userState.Userinfo = userinfo
		userState.ContactGroups = groupsResponse.ContactGroups
		userCache.Set(sessionID, userState)

		s.Values["id"] = fmt.Sprint(sessionID)
		if err := s.Save(c.Request(), c.Response()); err != nil {
			return err
		}

		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}).Name = "redirectURL"

	redirectURL := url.URL{Scheme: "http", Host: listenAddress, Path: e.Reverse("redirectURL")}
	googleOauthConfig = &oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Scopes:       []string{people.ContactsReadonlyScope, people.UserinfoEmailScope},
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURL.String(),
	}

	e.Logger.Fatal(e.Start(listenAddress))
}
