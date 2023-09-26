package html

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"time"

	g "github.com/maragudk/gomponents"
	c "github.com/maragudk/gomponents/components"

	//lint:ignore ST1001 follows gomponents convention
	. "github.com/maragudk/gomponents/html"

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
	FontAwesomeFS    = mustSub(fontAwesomeEmbed, "fontawesome-free-6.2.1-web")

	//go:embed tailwindcss-dist/*
	tailwindEmbed embed.FS
	TailwindFS    = mustSub(tailwindEmbed, "tailwindcss-dist")
)

func mustSub(emb fs.FS, dir string) fs.FS {
	fa, err := fs.Sub(emb, dir)
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
	WelcomeName          string
	LoginURL             string
	ClientID             string
	Groups               []*people.ContactGroup
	TableResults         []cohabitaters.XmasCard
	SelectedResourceName string
	GroupErrorMsg        string
	CountContacts        int
}

func ErrorPage(title, currentPath string) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    title,
		Language: "en",
		Head: []g.Node{
			Script(Src("https://unpkg.com/htmx.org@1.9.4"), g.Attr("integrity", "sha384-zUfuhFKKZCbHTY6aRR46gxiqszMk5tcHjsVFxnUo8VMus4kHGVdIYVbOYYNlKmHV"), g.Attr("crossorigin", "anonymous")),
			Script(Src("https://unpkg.com/hyperscript.org@0.9.11")),
			Link(Href("/static/tailwindcss/output.css"), Rel("stylesheet")),
			Link(Href("https://unpkg.com/flowbite@1.5.4/dist/flowbite.min.css"), Rel("stylesheet")),
			Link(Href("/static/fontawesome/css/fontawesome.css"), Rel("stylesheet")),
			Link(Href("/static/fontawesome/css/brands.css"), Rel("stylesheet")),
			Link(Href("/static/fontawesome/css/solid.css"), Rel("stylesheet")),
		},
		Body: []g.Node{
			Div(
				Class("flex min-h-screen w-full flex-col grow word-break"),
				Navbar(),
				Div(
					Class("flex flex-auto"),
					Div(
						Class("w-full p-2 bg-white"),
						P(g.Text("Oops.")),
						g.Text("Something went wrong"),
					),
				),
				Script(Src("https://unpkg.com/flowbite@1.5.4/dist/flowbite.js")),
			),
			MyFooter(),
		},
	})
}

func Navbar() g.Node {
	return Nav(
		Class("bg-gray-100 border-gray-200 px-2 sm:px-4 py-2.5 rounded dark:bg-gray-900"),
		Div(
			Class("container flex flex-wrap items-center justify-between mx-auto"),
			A(Href("/"), Class("flex items-center"),
				I(Class("fa-solid fa-people-arrows p-2")),
				Span(Class("self-center text-xl font-semibold whitespace-nowrap dark:text-black"),
					g.Text("Cohabitaters"),
				),
			),
			Div(
				Class("items-center justify-between hidden w-full md:flex md:w-auto md:order-1"), ID("navbar-cta"),
				Ul(Class("flex flex-col p-4 mt-4 border border-gray-100 rounded-lg bg-gray-100 md:flex-row md:space-x-8 md:mt-0 md:text-sm md:font-medium md:border-0 md:bg-gray-100 dark:bg-gray-800 md:dark:bg-gray-900 dark:border-gray-700"),
					NavbarItem("/auth/google/logout", "Logout"),
					NavbarItem("/about", "About"),
				),
			),
		),
	)
}

func NavbarItem(ref, text string) g.Node {
	return Li(A(
		Href(ref),
		Class("block py-2 pl-3 pr-4 text-gray-700 rounded hover:bg-gray-100 md:hover:bg-transparent md:hover:text-blue-700 md:p-0 md:dark:hover:text-white dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-white md:dark:hover:bg-transparent dark:border-gray-700"),
		g.Text(text),
	))
}

func MyFooter() g.Node {
	now := time.Now()
	return Footer(
		Class("p-4 bg-gray-100 rounded-lg shadow md:flex md:items-center md:justify-center md:p-6 dark:bg-gray-800"),
		Span(
			Class("text-sm text-gray-500 sm:text-center dark:text-gray-400"),
			g.Text(fmt.Sprintf("© %d ", now.Year())),
			A(Href("https://bfallik.net/"), Class("hover:underline"), g.Text("Brian Fallik")),
			g.Text(". All Rights Reserved."),
		),
	)
}
