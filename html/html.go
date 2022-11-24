package html

import (
	"embed"
	"html/template"
)

var (
	//go:embed templates/*
	fs embed.FS

	Templates = mustParse(
		tc{name: "index.html", paths: []string{"templates/index.html", "templates/base.html"}},
		tc{name: "error.html", paths: []string{"templates/error.html", "templates/base.html"}},
	)
)

type tc struct {
	name  string
	paths []string
}

func mustParse(config ...tc) []*template.Template {
	res := []*template.Template{}
	for _, c := range config {
		t := template.Must(template.New(c.name).ParseFS(fs, c.paths...))
		res = append(res, t)
	}
	return res
}
