package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
)

var port int
var domain string

func main() {

	flag.IntVar(&port, "p", 8080, "Provide a port number")
	flag.StringVar(&domain, "d", "localhost:"+fmt.Sprint(port), "Provide a domain name")
	flag.Parse()

	r := chi.NewRouter()
	// r.Use(middleware.Logger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err := IndexView().Render(r.Context(), w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	r.Get("/new", NewChatHandler)
	r.Get("/end", EndChatHandler)

	r.Route("/c/{chatId}", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(ChatMiddleware)
			r.Get("/", ChatHandler)
			r.Post("/", PostMessageHandler)
			r.Get("/sse", ReceiveMessageHandler)
		})

		r.Get("/status", ChatStatusHandler)
	})

	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "static"))
	FileServer(r, "/static", filesDir)

	http.ListenAndServe(":"+fmt.Sprint(port), r)
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
