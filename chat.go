package main

import (
	"fmt"
	"net/http"
	"strings"
)

type Chat struct {
	clients map[*Client]bool
}

func (c *Chat) broadcast(message Message) {
	for client := range c.clients {
		client.receive <- message
	}
}

type Client struct {
	id      string
	name    string
	receive chan Message
}

func (c *Client) listen(w http.ResponseWriter, r *http.Request) {
	for {
		select {
		case <-r.Context().Done():
			fmt.Println("Client disconnected:", c.id)
			return
		case message := <-c.receive:
			fmt.Println("Sending message:", message.text)
			message.sendEvent(w, r)
		}
		w.(http.Flusher).Flush()
	}
}

type Message struct {
	text string
	// author *Client
}

func (m *Message) sendEvent(w http.ResponseWriter, r *http.Request) error {
	sb := &strings.Builder{}

	sb.WriteString("event: message\ndata: ")

	if err := MessageView(m).Render(r.Context(), sb); err != nil {
		return err
	}

	sb.WriteString("\n\n")

	_, err := fmt.Fprint(w, sb.String())
	return err
}
