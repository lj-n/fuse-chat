package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/uuid"
)

type Chat struct {
	conns map[string]chan Message
}

func (c *Chat) broadcast(message Message) {
	for _, conn := range c.conns {
		conn <- message
	}
}

type Message struct {
	text string
}

func (m *Message) render(ctx context.Context) (string, error) {
	buf := &strings.Builder{}
	err := MessageView(m).Render(ctx, buf)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

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
			conns: make(map[string]chan Message),
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

			clientId := uuid.New().String()
			conn := make(chan Message)
			chat.conns[clientId] = conn

			for data := range conn {

				fmt.Println("Sending message to client:", data.text)

				sb := strings.Builder{}

				message, err := data.render(r.Context())
				if err != nil {
					fmt.Println("Error rendering message:", err)
					break
				}

				sb.WriteString(fmt.Sprintf("event: %s\n", "message"))
				sb.WriteString(fmt.Sprintf("data: %v\n\n", message))

				_, err = fmt.Fprint(w, sb.String())
				if err != nil {
					fmt.Println("Error writing to client:", err)
					break
				}

				flusher.Flush()
			}
		})
	})

	http.ListenAndServe(":5173", r)
}
