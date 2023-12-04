package main

import (
	"log"
)

type Message struct {
	From *Client
	Data []byte
}

func NewMessage(client *Client, data []byte) *Message {
	return &Message{
		From: client,
		Data: data,
	}
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 10),
		register:   make(chan *Client, 10),
		unregister: make(chan *Client, 10),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf(
				"Hub registered<%d>(%s) | %v\n",
				client.Id,
				client.Conn.RemoteAddr().String(),
				h.clients,
			)

		case client := <-h.unregister:
			delete(h.clients, client)
			client.Close()
			log.Printf(
				"Hub unregistered<%d>(%s) | %v\n",
				client.Id,
				client.Conn.RemoteAddr().String(),
				h.clients,
			)

		case msg := <-h.broadcast:
			log.Printf(
				"Hub broadcasting: `%s`\n",
				string(msg.Data),
			)
			for client := range h.clients {
				if client == msg.From {
					continue
				}
				log.Printf("Hub broadcasting to client<%d>\n", client.Id)
				select {
				case client.Send <- msg.Data:
				default:
					client.Conn.Close()
				}
			}
		}
	}
}
