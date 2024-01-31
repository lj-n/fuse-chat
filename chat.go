package main

import (
	"fmt"
	"net/http"
	"strings"
)

type Chat struct {
	id    string
	conns map[string]*Connection
}

func (c *Chat) broadcast(message Message) {
	for _, connection := range c.conns {
		connection.receive <- message
	}
}

type Connection struct {
	client  *Client
	receive chan Message
}

type Client struct {
	id   string
	name string
}

type Message struct {
	text   string
	client *Client
}

func (m *Message) sendEvent(w http.ResponseWriter, r *http.Request, isAuthor bool) error {
	sb := &strings.Builder{}

	sb.WriteString("event: message\ndata: ")

	if err := MessageView(m, isAuthor).Render(r.Context(), sb); err != nil {
		return err
	}

	sb.WriteString("\n\n")

	_, err := fmt.Fprint(w, sb.String())
	return err
}

var chats = make(map[string]*Chat)
