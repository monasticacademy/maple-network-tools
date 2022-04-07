package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

//go:embed static/favicon.ico static/favicon-16x16.png static/favicon-32x32.png static/apple-touch-icon.png
var assets embed.FS

// handleSpecialAsset handles top-level assets like /favicon.ico that are stored in the static dir
func (a *app) handleSpecialAsset(w http.ResponseWriter, r *http.Request) {
	fs := http.FileServer(http.FS(assets))
	r.URL.Path = "static" + r.URL.Path
	fs.ServeHTTP(w, r)
}

func (a *app) runWebUI(ctx context.Context, port string) {
	// create file server for static assets
	fs := http.FileServer(http.FS(assets))

	for _, path := range []string{"favicon.ico", "/favicon.ico", "static/favicon.ico", "/static/favicon.ico"} {
		_, err := assets.Open(path)
		fmt.Printf("%s: %v\n", path, err)
	}

	// set up the routes
	http.HandleFunc("/favicon.ico", a.handleSpecialAsset)
	http.HandleFunc("/favicon-16x16.png", a.handleSpecialAsset)
	http.HandleFunc("/favicon-32x32.png", a.handleSpecialAsset)
	http.HandleFunc("/apple-touch-icon.png", a.handleSpecialAsset)
	http.Handle("/static/", http.StripPrefix("/", fs))
	http.HandleFunc("/", a.handleRoot)

	// start the http server
	log.Println("listening on " + port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func f(w io.Writer, label, err string, rtt int64) {
	if err == "" {
		fmt.Fprintf(w, "%-10s %10s (%v)\n", label, "OK", time.Microsecond*time.Duration(rtt))
	} else {
		fmt.Fprintf(w, "%-10s %10s (%v)\n", label, "FAIL", err)
	}
}

func (a *app) handleRoot(w http.ResponseWriter, r *http.Request) {
	latest := a.latest()
	if len(latest) == 0 {
		fmt.Fprintln(w, "no ping records in buffer")
		return
	}

	f(w, "modem", latest[0].GetModemError(), latest[0].GetModemLatency())
	f(w, "router", latest[0].GetRouterError(), latest[0].GetRouterLatency())
	f(w, "google", latest[0].GetGoogleError(), latest[0].GetGoogleLatency())
}
