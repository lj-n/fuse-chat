package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	text      string
	client    *Client
	createdAt time.Time
}

// SendServerEvent sends a server event containing the message data to the client.
// It writes the event data to the provided http.ResponseWriter and returns an error if any.
// The isAuthor parameter indicates whether the current user is the author of the message.
func (m *Message) SendServerEvent(w http.ResponseWriter, r *http.Request, isAuthor bool) error {
	sb := &strings.Builder{}
	sb.WriteString("event: message\ndata: ")

	// Render the message data using the MessageView and write it to the strings.Builder
	if err := MessageView(m, isAuthor).Render(r.Context(), sb); err != nil {
		return err
	}

	sb.WriteString("\n\n")
	_, err := fmt.Fprint(w, sb.String())
	return err
}

// postMessageHandler handles the HTTP POST request for posting a message.
// It receives the message from the request form and creates a new Message object.
// The message is then passed to the chat's ReceiveMessage method.
// Finally, it sets the HTTP status code to 204 (No Content) to indicate success.
func PostMessageHandler(w http.ResponseWriter, r *http.Request) {
	chat := r.Context().Value(ContextChatKey).(*Chat)
	client := r.Context().Value(ContextClientKey).(*Client)

	message := &Message{
		text:      r.FormValue("message"),
		client:    client,
		createdAt: time.Now(),
	}

	chat.ReceiveMessage(message)

	w.WriteHeader(http.StatusNoContent)
}

// receiveMessageHandler handles the HTTP request for receiving messages.
// It sets up the server-sent event (SSE) response and continuously sends messages to the client.
// The SSE response is flushed after each message is sent.
// If SSE is not supported, it returns an internal server error.
// The connection is tracked using a unique connection ID.
func ReceiveMessageHandler(w http.ResponseWriter, r *http.Request) {
	chat := r.Context().Value(ContextChatKey).(*Chat)
	client := r.Context().Value(ContextClientKey).(*Client)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	connectionId := uuid.New().String()
	connection := &Connection{
		client:  client,
		receive: make(chan *Message),
	}
	chat.conns[connectionId] = connection

	_, cancel := context.WithCancel(r.Context())
	defer func() {
		delete(chat.conns, connectionId)
		cancel()
	}()

loop:
	for {
		select {
		case <-r.Context().Done():
			break loop
		case message := <-connection.receive:
			if err := message.SendServerEvent(w, r, client.Id == message.client.Id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			flusher.Flush()
		}
	}
}
