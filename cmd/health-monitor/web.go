package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/dustin/go-humanize"
)

//go:embed static
var assets embed.FS

//go:embed status.template.html
var statusRaw []byte

// parse the template just once, at program startup
var statusTemplate = template.Must(template.New("status").Funcs(template.FuncMap{
	"since": func(t int64) string {
		return humanize.Time(time.UnixMicro(t))
	},
	"seconds": func(d int64) string {
		return (time.Duration(d) * time.Microsecond).String()
	},
}).Parse(string(statusRaw)))

// handleSpecialAsset handles top-level assets like /favicon.ico that are stored in the static dir
func (a *app) handleSpecialAsset(w http.ResponseWriter, r *http.Request) {
	fs := http.FileServer(http.FS(assets))
	r.URL.Path = "static" + r.URL.Path
	fs.ServeHTTP(w, r)
}

func (a *app) runWebUI(ctx context.Context, port string) {
	// create file server for static assets
	fs := http.FileServer(http.FS(assets))

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

// func writeLine(w io.Writer, label, err string, rtt int64) {
// 	if err == "" {
// 		fmt.Fprintf(w, "%-40s %10s (%v)\n", label, "OK", time.Microsecond*time.Duration(rtt))
// 	} else {
// 		fmt.Fprintf(w, "%-40s %10s (%v)\n", label, "FAIL", err)
// 	}
// }

// Payload for the template
type htmlPayload struct {
	Checks []*HealthCheck
}

func (a *app) handleRoot(w http.ResponseWriter, r *http.Request) {
	latest := a.latest()
	if len(latest) == 0 {
		fmt.Fprintln(w, "no ping records in buffer")
		return
	}

	err := statusTemplate.Execute(w, htmlPayload{
		Checks: latest[0],
	})
	if err != nil {
		msg := fmt.Sprintf("error executing template: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// var ts time.Time
	// for _, c := range latest[0] {
	// 	writeLine(w, c.Operation, c.Error, c.Duration)
	// 	ts = time.UnixMicro(c.Timestamp)
	// }

	// fmt.Fprintf(w, "\nas of %v ago\n", time.Since(ts))
}
