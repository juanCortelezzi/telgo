package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"time"
	"unicode"
)

type Client struct {
	Name       string
	Authorized bool
	Conn       *net.TCPConn
	Send       chan []byte
	Recv       chan []byte
}

func NewClient(conn *net.TCPConn) *Client {
	return &Client{
		Name:       "",
		Authorized: false,
		Conn:       conn,
		Send:       make(chan []byte),
		Recv:       make(chan []byte),
	}
}

func (client *Client) Run(ctx context.Context, hub *Hub) {
	go writePump(ctx, client)
	go interceptor(ctx, client, hub)
	readPump(ctx, client, hub)
}

func (client *Client) Close() {
	close(client.Send)
	close(client.Recv)
	client.Conn.Close()
}

func isLetterRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

func isAsciiRune(r rune) bool {
	return r <= unicode.MaxASCII
}

func isValidString(msg []byte, isValidRune func(r rune) bool) bool {
	for _, r := range bytes.Runes(msg) {
		if !isValidRune(r) {
			return false
		}
	}
	return true
}

func readPump(ctx context.Context, client *Client, hub *Hub) {
	logger := LoggerFromContext(ctx)
	defer (func() { hub.unregister <- client })()

	msgBuffer := make([]byte, 1024)
	for {
		pointer, err := client.Conn.Read(msgBuffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logger.Error("Failed to read from connection", "clientName", client.Name, "error", err)
			}
			return
		}

		msg := bytes.TrimSpace(msgBuffer[:pointer])
		if len(msg) == 0 {
			continue
		}

		if !isValidString(msg, isAsciiRune) {
			logger.Warn("Invalid message from client", "message", string(msg))
			continue
		}

		logger.Debug("Received message from client", "messageLen", len(msg), "message", string(msg))

		select {
		case client.Recv <- msg:
		case <-time.After(time.Second * 1):
			logger.Warn("Dropped message from client", "message", string(msg))
		}
	}
}

func writePump(ctx context.Context, client *Client) {
	logger := LoggerFromContext(ctx)
	for msg := range client.Send {
		_, err := client.Conn.Write(msg)
		if err != nil {
			logger.Error("Failed to write to connection", "error", err)
			return
		}
	}
}

func interceptor(ctx context.Context, client *Client, hub *Hub) {
	logger := LoggerFromContext(ctx)

	for data := range client.Recv {
		logger.Debug("Intercepted message from client", "messageBytes", data)

		if client.Authorized {
			hub.broadcast <- NewMessage(client, data)
			continue
		}

		rawName, found := bytes.CutPrefix(data, []byte(":connect "))
		if !found {
			client.Send <- []byte("Oi, provide a name!\n:connect <your name>\n")
			continue
		}

		name := bytes.TrimSpace(rawName)
		if !isValidString(name, isLetterRune) {
			client.Send <- []byte(
				"Oi, provide a valid name (full ascii)!\n:connect <your name>\n",
			)
			continue
		}

		client.Authorized = true
		client.Name = string(name)
		client.Send <- []byte("Connected!\n")

		hub.register <- client
	}
}
