package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/google/uuid"
)

type Client struct {
	Id   string `json:"id,string"`
	Name string `json:"name,string"`
}

const (
	cookieName string = "fuse_chat_cookie"
)

// parseClientCookie parses the client cookie from the given HTTP request.
// It decodes the cookie value, unmarshals it into a Client struct,
// and validates the client ID using UUID parsing.
// If successful, it returns the parsed Client object.
// Otherwise, it returns an error.
func parseClientCookie(r *http.Request) (*Client, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, err
	}

	client := &Client{}

	err = json.Unmarshal(data, client)
	if err != nil {
		return nil, err
	}

	_, err = uuid.Parse(client.Id)
	if err != nil {
		return nil, err
	}

	return client, nil

}

// newClient creates a new client for the chat application.
// It takes in the http.ResponseWriter, http.Request, and a Chat instance.
// If the client cookie exists in the request, it parses the client information from the cookie.
// If the client cookie does not exist, it generates a new client with a unique ID and a random name.
// The client information is then stored in a cookie and set in the response.
// The function returns the created client.
func newClient(w http.ResponseWriter, r *http.Request, c *Chat) *Client {
	client, err := parseClientCookie(r)

	if err != nil {
		client = &Client{
			Id:   uuid.New().String(),
			Name: petname.Generate(2, "-"),
		}

		data, _ := json.Marshal(client)

		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    base64.StdEncoding.EncodeToString(data),
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	return client
}
