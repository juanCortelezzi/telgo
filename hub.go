package main

import (
	"bytes"
	"context"
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

func (h *Hub) broadcastMessage(ctx context.Context, msg *Message) {
	logger := LoggerFromContext(ctx)

	_, isClientRegistered := h.clients[msg.From]
	if !isClientRegistered {
		logger.Error(
			"Trying to broadcast message from unregistered client",
			"clientName", msg.From.Name,
		)
		return
	}

	logger.Debug("Broadcasting", "message", string(msg.Data))

	prefix := []byte(msg.From.Name + ": ")
	msgBytes := bytes.Join([][]byte{prefix, msg.Data, {'\n'}}, []byte{})

	for client := range h.clients {
		if client == msg.From {
			continue
		}

		select {
		case client.Send <- msgBytes:
		default:
			delete(h.clients, client)
			client.Close()
			logger.Warn(
				"Client unregistered because to send chan is full",
				"ip", client.Conn.RemoteAddr().String(),
				"name", client.Name,
				"allClients", h.clients,
			)
		}
	}
}

func (h *Hub) registerClient(ctx context.Context, client *Client) {
	logger := LoggerFromContext(ctx)
	h.clients[client] = true
	logger.Debug(
		"Client registered",
		"ip", client.Conn.RemoteAddr().String(),
		"name", client.Name,
		"allClients", h.clients,
	)
}

func (h *Hub) unregisterClient(ctx context.Context, client *Client) {
	logger := LoggerFromContext(ctx)
	delete(h.clients, client)
	client.Close()
	logger.Debug(
		"Client unregistered",
		"ip", client.Conn.RemoteAddr().String(),
		"name", client.Name,
		"allClients", h.clients,
	)
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.registerClient(ctx, client)

		case client := <-h.unregister:
			h.unregisterClient(ctx, client)

		case msg := <-h.broadcast:
			h.broadcastMessage(ctx, msg)
		}
	}
}
