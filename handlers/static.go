package handlers

import (
	"net/http"

	"github.com/bfallik/cohabitaters/html"
	"github.com/labstack/echo/v4"
)

var FontAwesome = echo.WrapHandler(
	http.StripPrefix("/static/fontawesome/",
		http.FileServer(http.FS(html.FontAwesomeFS))))

var Tailwind = echo.WrapHandler(
	http.StripPrefix("/static/tailwindcss/",
		http.FileServer(http.FS(html.TailwindFS))))
