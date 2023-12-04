package main

import (
	"log"
	"strings"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 10),
		register:   make(chan *Client, 10),
		unregister: make(chan *Client, 10),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			log.Printf(
				"Hub registering<%d>(%s)\n",
				client.id,
				client.conn.RemoteAddr().String(),
			)
			h.clients[client] = true

		case client := <-h.unregister:
			log.Printf(
				"Hub unregistering<%d>(%s) | %v\n",
				client.id,
				client.conn.RemoteAddr().String(),
				h.clients,
			)
			delete(h.clients, client)
			close(client.send)
			client.conn.Close()

		case msg := <-h.broadcast:
			log.Printf("Hub broadcasting: `%s`\n", strings.TrimSpace(string(msg)))
			for client := range h.clients {
				log.Printf("Hub broadcasting to client<%d>\n", client.id)
				select {
				case client.send <- msg:
				default:
					client.conn.Close()
				}
			}
		}
	}
}
