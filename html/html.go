package html

import (
	"embed"
	"io"
	"text/template"
)

var (
	//go:embed templates/*
	indexFS embed.FS
	index   = parse("templates/index.html")
)

func Index(w io.Writer, data any) error {
	return index.Execute(w, data)
}

func parse(file string) *template.Template {
	return template.Must(
		template.New("index.html").ParseFS(indexFS, file))
}
