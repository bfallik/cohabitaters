package main

import (
	"log"
	"net"
	"net/http"

	"github.com/bfallik/cohabitaters/html"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := html.Index(w, nil); err != nil {
			panic(err.Error())
		}
	})

	server := &http.Server{
		Addr:    net.JoinHostPort("", "8080"),
		Handler: mux,
	}

	log.Printf("starting HTTP server at %q", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	} else {
		log.Println("server closed")
	}
}
