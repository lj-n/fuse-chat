package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

const (
	ChatDuration time.Duration = time.Duration(1) * time.Minute
)

var chats = make(map[string]*Chat)

type Chat struct {
	id         string
	conns      map[string]*Connection
	createTime time.Time
	endTime    time.Time
	duration   time.Duration
	messages   []Message
}

type Connection struct {
	client  *Client
	receive chan *Message
}

func (c *Chat) TimeRemaining() string {
	remaining := time.Until(c.endTime)
	return fmt.Sprintf("Time left: %s", remaining.Round(time.Second))
}

func (c *Chat) Connections() string {
	return fmt.Sprintf("Connections: %d", len(c.conns))
}

func (c *Chat) Age() string {
	age := time.Since(c.createTime)
	return fmt.Sprintf("Chat age: %s", age.Round(time.Second))
}

func (c *Chat) URL() string {
	return fmt.Sprintf("%s/c/%s", domain, c.id)
}

// ReceiveMessage broadcasts the given message to all connected clients and adds it to the chat's message history.
// It also resets the chat's fuse by updating the end time.
func (c *Chat) ReceiveMessage(m *Message) {
	for _, connection := range c.conns {
		connection.receive <- m
	}
	c.messages = append(c.messages, *m)
	c.endTime = time.Now().Add(c.duration)
}

// StartFuse starts the "fuse" for the chat.
// It continuously checks if the time until the chat's end time has elapsed.
// If the end time has elapsed, it deletes the chat from the chats map.
func (c *Chat) StartFuse() {
	for {
		if time.Until(c.endTime) <= 0 {
			delete(chats, c.id)
			break
		}
	}
}

// NewChat creates a new Chat instance with the given duration.
// It generates a unique ID using UUID and initializes the chat's properties.
// The chat is added to the chats map and its fuse is started in a separate goroutine.
// Returns the created Chat instance.
func NewChat(d time.Duration) *Chat {
	chat := &Chat{
		id:         uuid.New().String(),
		conns:      make(map[string]*Connection),
		createTime: time.Now(),
		endTime:    time.Now().Add(d),
		duration:   d,
		messages:   make([]Message, 0),
	}

	chats[chat.id] = chat
	go chat.StartFuse()
	return chat
}

type key string

const (
	// ContextChatKey is the key used to store the chat context in a request context.
	ContextChatKey key = "chat"

	// ContextClientKey is the key used to store the client context in a request context.
	ContextClientKey key = "client"
)

// ChatMiddleware is a middleware function that handles request on the /c/ route.
// It retrieves the chat ID from the URL parameters, checks if the chat exists,
// creates a new client, and adds the chat and client to the request context.
// The next handler is then called with the updated request context.
func ChatMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paramId := chi.URLParam(r, "chatId")
		chat, ok := chats[paramId]
		if !ok {
			http.NotFound(w, r)
			return
		}

		// get or create new client
		client := newClient(w, r, chat)

		// add chat and client to handler context
		ctx := context.WithValue(r.Context(), ContextChatKey, chat)
		ctx = context.WithValue(ctx, ContextClientKey, client)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// NewChatHandler is a handler function that creates a new chat and redirects the user to the chat page.
func NewChatHandler(w http.ResponseWriter, r *http.Request) {
	chat := NewChat(ChatDuration)
	http.Redirect(w, r, "/c/"+chat.id, http.StatusFound)
}

// ChatHandler handles the HTTP request for the chat functionality.
// It retrieves the chat from the request context and renders the ChatView template.
// If there is an error during rendering, it returns an internal server error.
func ChatHandler(w http.ResponseWriter, r *http.Request) {
	chat := r.Context().Value(ContextChatKey).(*Chat)
	client := r.Context().Value(ContextClientKey).(*Client)

	err := ChatView(chat, client).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// EndChatHandler is a HTTP handler function that renders the ChatEndView template.
func EndChatHandler(w http.ResponseWriter, r *http.Request) {
	err := ChatEndView().Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ChatStatusHandler handles the HTTP request for retrieving the status of a chat.
// It takes the chat ID from the URL parameter and checks if the chat exists.
// If the chat does not exist, it sets the "HX-Redirect" header to "/end" and returns a 286 status code (to end htmx polling).
// If the chat exists, it renders the ChatStatusView using the chat data and writes the response.
// This handler does not use the chat middleware.
func ChatStatusHandler(w http.ResponseWriter, r *http.Request) {
	chatId := chi.URLParam(r, "chatId")
	chat, ok := chats[chatId]
	if !ok {
		w.Header().Add("HX-Redirect", "/end")
		w.WriteHeader(286)
		return
	}

	err := ChatStatusView(chat).Render(r.Context(), w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
