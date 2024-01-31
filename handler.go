package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

type key string

const (
	ctxChatKey   key    = "chat"
	ctxClientKey key    = "client"
	cookieName   string = "client_cookie"
)

func getClient(w http.ResponseWriter, r *http.Request, c *Chat) *Client {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		// create new client
		clientId := uuid.New().String()
		client := &Client{
			id:   clientId,
			name: "Anonymous",
		}

		// set cookie
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    clientId,
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
		fmt.Println("Created new client and set cookie: ", client.id)
		return client
	}

	// create client from cookie
	return &Client{
		id:   cookie.Value,
		name: "Anonymous",
	}

}

func chatMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check if chat exists
		chatId := chi.URLParam(r, "chatId")
		chat, ok := chats[chatId]
		if !ok {
			fmt.Println("Chat not found: ", chatId)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// get or create new client
		client := getClient(w, r, chat)

		// add chat and client to handler context
		ctx := context.WithValue(r.Context(), ctxChatKey, chat)
		ctx = context.WithValue(ctx, ctxClientKey, client)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := IndexView().Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newChatHandler(w http.ResponseWriter, r *http.Request) {
	id := uuid.New().String()

	chat := &Chat{
		id:    id,
		conns: make(map[string]*Connection),
	}
	chats[id] = chat

	http.Redirect(w, r, "/c/"+id, http.StatusFound)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	chat := r.Context().Value(ctxChatKey).(*Chat)

	err := ChatView(chat.id).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func chatMessageHandler(w http.ResponseWriter, r *http.Request) {
	chat := r.Context().Value(ctxChatKey).(*Chat)
	client := r.Context().Value(ctxClientKey).(*Client)

	message := r.FormValue("message")
	chat.broadcast(Message{
		text:   message,
		client: client,
	})

	w.WriteHeader(http.StatusNoContent)
}

func chatSSEHandler(w http.ResponseWriter, r *http.Request) {
	chat := r.Context().Value(ctxChatKey).(*Chat)
	client := r.Context().Value(ctxClientKey).(*Client)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	fmt.Println("Client connected:", client.id)
	connectionId := uuid.New().String()
	connection := &Connection{
		client:  client,
		receive: make(chan Message),
	}
	chat.conns[connectionId] = connection

	_, cancel := context.WithCancel(r.Context())
	defer func() {
		fmt.Println("Client disconnected: ", client.id)
		delete(chat.conns, connectionId)
		cancel()
	}()

loop:
	for {
		select {
		case <-r.Context().Done():
			break loop
		case message := <-connection.receive:
			fmt.Println("Sending message:", message.text)
			message.sendEvent(w, r, client.id == message.client.id)
			flusher.Flush()
		}
	}
}
