package main

import (
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"

	"github.com/bfallik/cohabitaters/html"
	"github.com/labstack/echo/v4"
)

type Templater struct {
	templates map[string]*template.Template
}

func newTemplater(tmpls ...*template.Template) *Templater {
	m := map[string]*template.Template{}
	for _, t := range tmpls {
		m[t.Name()] = t
	}
	return &Templater{templates: m}
}

func (t *Templater) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}
	return tmpl.ExecuteTemplate(w, "base", data)
}

func main() {
	e := echo.New()

	e.Renderer = newTemplater(html.Templates...)

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", nil)
	})

	e.GET("/error", func(c echo.Context) error {
		return c.Render(http.StatusInternalServerError, "error.html", nil)
	})

	e.Logger.Fatal(e.Start(net.JoinHostPort("", "8080")))
}
