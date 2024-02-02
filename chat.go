package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

var chats = make(map[string]*Chat)

const fuseTime = time.Duration(2) * time.Minute

type Chat struct {
	id         string
	conns      map[string]*Connection
	createTime time.Time
	endTime    time.Time
	fuse       time.Timer
}

func (c *Chat) broadcast(message Message) {
	for _, connection := range c.conns {
		connection.receive <- message
	}
}

func (c *Chat) startFuse() {
	<-c.fuse.C
	fmt.Println("Chat fuse blown:", c.id)
	for _, connection := range c.conns {
		connection.fuseEnd <- true
	}
	delete(chats, c.id)
}

func (c *Chat) resetFuse() {
	c.fuse.Reset(fuseTime)
	c.endTime = time.Now().Add(fuseTime)
}

func (c *Chat) TimeRemaining() string {
	remaining := time.Until(c.endTime)
	seconds := int(remaining.Seconds())
	return fmt.Sprintf("Seconds left: %d", seconds)
}

func (c *Chat) ConnectionStatus() string {
	return fmt.Sprintf("Connections: %d", len(c.conns))
}

func (c *Chat) URL() string {
	// TODO: This is a hack, we should use the router to generate the URL
	prefix := "http://localhost:5173"
	return fmt.Sprintf("%s/c/%s", prefix, c.id)
}

func newChat() *Chat {
	id := uuid.New().String()

	chat := &Chat{
		id:         id,
		conns:      make(map[string]*Connection),
		createTime: time.Now(),
		endTime:    time.Now().Add(fuseTime),
		fuse:       *time.NewTimer(fuseTime),
	}
	chats[id] = chat

	go chat.startFuse()

	return chat
}

type Connection struct {
	client  *Client
	receive chan Message
	fuseEnd chan bool
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
