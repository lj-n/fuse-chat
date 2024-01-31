package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
)

var chats = make(map[string]*Chat)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		err := IndexView().Render(r.Context(), w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Get("/new", func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New()

		chat := &Chat{
			clients: make(map[*Client]bool),
		}
		chats[id.String()] = chat

		http.Redirect(w, r, "/c/"+id.String(), http.StatusFound)
	})

	r.Route("/c/{chatId}", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			chatId := chi.URLParam(r, "chatId")
			if _, ok := chats[chatId]; !ok {
				http.NotFound(w, r)
				return
			}

			err := ChatView(chatId).Render(r.Context(), w)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			chatId := chi.URLParam(r, "chatId")

			chat, ok := chats[chatId]

			if !ok {
				http.NotFound(w, r)
				return
			}

			message := r.FormValue("message")
			fmt.Println("Recieved message:", message)
			chat.broadcast(Message{message})

			w.WriteHeader(http.StatusNoContent)
		})

		r.Get("/sse", func(w http.ResponseWriter, r *http.Request) {
			chatId := chi.URLParam(r, "chatId")
			chat, ok := chats[chatId]
			if !ok {
				http.NotFound(w, r)
				return
			}

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "SSE not supported", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			_, cancel := context.WithCancel(r.Context())
			defer func() {
				fmt.Println("Client disconnected")
				cancel()
			}()

			client := &Client{
				id:      uuid.New().String(),
				receive: make(chan Message),
				name:    "Anonymous",
			}

			chat.clients[client] = true

			for message := range client.receive {
				err := message.sendView(w, r)
				if err != nil {
					fmt.Println("Error sending message to client:", err)
					break
				}

				flusher.Flush()
			}
		})
	})

	http.ListenAndServe(":5173", r)
}
