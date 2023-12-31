package main

import (
	"bytes"
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
				"Hub registered<%s>(%s) | %v\n",
				client.Conn.RemoteAddr().String(),
				client.Name,
				h.clients,
			)

		case client := <-h.unregister:
			delete(h.clients, client)
			client.Close()
			log.Printf(
				"Hub unregistered<%s>(%s) | %v\n",
				client.Conn.RemoteAddr().String(),
				client.Name,
				h.clients,
			)

		case msg := <-h.broadcast:
			log.Printf("Hub broadcasting: `%v`\n", string(msg.Data))

			prefix := []byte(msg.From.Name + ": ")

			for client := range h.clients {
				if client == msg.From {
					continue
				}

				select {
				case client.Send <- bytes.Join(
					[][]byte{prefix, msg.Data, {'\n'}},
					[]byte{},
				):
				default:
					client.Conn.Close()
				}
			}
		}
	}
}
