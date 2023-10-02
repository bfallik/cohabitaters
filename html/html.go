package html

import (
	"embed"
	"io/fs"

	"github.com/a-h/templ"
	"github.com/bfallik/cohabitaters/html/templs"
)

var (
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

type TmplIndexData = templs.PageIndexInput

func ComponentPageIndex(input TmplIndexData) templ.Component {
	return templs.PageIndex(input)
}

func ComponentTableResults(input TmplIndexData) templ.Component {
	return templs.Results(input)
}
