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

type Message struct {
	text string
	// author *Client
}

func (m *Message) sendView(w http.ResponseWriter, r *http.Request) error {

	cb := &strings.Builder{}

	if err := MessageView(m).Render(r.Context(), cb); err != nil {
		return err
	}

	sb := &strings.Builder{}

	sb.WriteString("event: message\n")
	sb.WriteString(fmt.Sprintf("data: %v\n\n", cb.String()))

	_, err := fmt.Fprint(w, sb.String())
	return err
}
