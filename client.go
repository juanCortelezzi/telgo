package main

import (
	"bytes"
	"errors"
	"io"
	"log"
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

func ReadPump(client *Client, hub *Hub) {
	defer (func() { hub.unregister <- client })()

	msgBuffer := make([]byte, 1024)
	for {
		pointer, err := client.Conn.Read(msgBuffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("Failed to read from connection(%s): %v\n", client.Name, err)
			}
			return
		}

		msg := bytes.TrimSpace(msgBuffer[:pointer])
		if len(msg) == 0 {
			continue
		}

		if !isValidString(msg, isAsciiRune) {
			log.Printf("Invalid message from client: `%s`\n", string(msg))
			continue
		}

		log.Printf("Received message from client<%d>(%s)\n", len(msg), string(msg))

		select {
		case client.Recv <- msg:
		case <-time.After(time.Second * 1):
			log.Printf("Dropped message from client: %s\n", string(msg))
		}
	}
}

func WritePump(client *Client) {
	for msg := range client.Send {
		_, err := client.Conn.Write(msg)
		if err != nil {
			log.Printf("Failed to write to connection: %v\n", err)
			return
		}
	}
}

func Interceptor(client *Client, hub *Hub) {
	for data := range client.Recv {
		log.Printf("Intercepted message from client\n")

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
