package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", indexHandler)
	r.Get("/new", newChatHandler)

	r.Route("/c/{chatId}", func(r chi.Router) {
		r.Use(chatMiddleware)
		r.Post("/", chatMessageHandler)
		r.Get("/", chatHandler)
		r.Get("/fuse", fuseHandler)
		r.Get("/sse", chatSSEHandler)
	})

	http.ListenAndServe(":5173", r)
}
