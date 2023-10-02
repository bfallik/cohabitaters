package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/bfallik/cohabitaters/html"
	"github.com/bfallik/cohabitaters/html/templs"
	"github.com/labstack/echo/v4"
)

var FontAwesome = echo.WrapHandler(
	http.StripPrefix("/static/fontawesome/",
		http.FileServer(http.FS(html.FontAwesomeFS))))

var Tailwind = echo.WrapHandler(
	http.StripPrefix("/static/tailwindcss/",
		http.FileServer(http.FS(html.TailwindFS))))

var Error = echo.WrapHandler(templ.Handler(templs.PageError(), templ.WithStatus(http.StatusInternalServerError)))

var About = echo.WrapHandler(templ.Handler(templs.PageAbout()))
