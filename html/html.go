package html

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"

	"github.com/bfallik/cohabitaters"
	"google.golang.org/api/people/v1"
)

var (
	//go:embed templates/*
	templatesEmbed embed.FS

	Templates = mustParse(
		tc{name: "index.html", paths: []string{"templates/index.html", "templates/partials/*.html"}},
		tc{name: "partials/results.html", paths: []string{"templates/partials/*.html"}},
		tc{name: "error.html", paths: []string{"templates/error.html", "templates/partials/*.html"}},
		tc{name: "about.html", paths: []string{"templates/about.html", "templates/partials/*.html"}},
	)

	//go:embed fontawesome-free-6.2.1-web/*
	fontAwesomeEmbed embed.FS

	FontAwesomeFS = mustSub(fontAwesomeEmbed)
)

func mustSub(emb fs.FS) fs.FS {
	fa, err := fs.Sub(emb, "fontawesome-free-6.2.1-web")
	if err != nil {
		panic(err)
	}
	return fa
}

type tc struct {
	name  string
	paths []string
}

func mustParse(config ...tc) []*template.Template {
	res := []*template.Template{}
	for _, c := range config {
		t := template.Must(template.New(c.name).ParseFS(templatesEmbed, c.paths...))
		res = append(res, t)
	}
	return res
}

type Templater struct {
	templates map[string]*template.Template
}

func NewTemplater(tmpls ...*template.Template) *Templater {
	m := map[string]*template.Template{}
	for _, t := range tmpls {
		m[t.Name()] = t
	}
	return &Templater{templates: m}
}

func (t *Templater) Render(w io.Writer, name string, data interface{}) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}
	return tmpl.ExecuteTemplate(w, name, data)
}

type TmplIndexData struct {
	WelcomeMsg           string
	Groups               []*people.ContactGroup
	TableResults         []cohabitaters.XmasCard
	SelectedResourceName string
}
