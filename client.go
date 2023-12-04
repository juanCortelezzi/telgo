package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"
	"unicode"
)

type ID = uint64

var ClientIDCounter atomic.Uint64

type Client struct {
	id   ID
	conn *net.TCPConn
	send chan []byte
	recv chan []byte
}

func isValidMessage(msg []byte) bool {
	for _, r := range bytes.Runes(msg) {
		if r <= 0 || r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func readPump(client *Client, hub *Hub) {
	defer (func() { hub.unregister <- client })()

	msgBuffer := make([]byte, 32)
	for {
		pointer, err := client.conn.Read(msgBuffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("Failed to read from connection: %v\n", err)
			}
			return
		}

		msg := bytes.TrimSpace(msgBuffer[:pointer])

		if !isValidMessage(msg) {
			log.Printf("Invalid message from client: `%s`\n", string(msg))
			continue
		}

		log.Printf("Received message from client: `%s`\n", string(msg))

		select {
		case <-time.After(time.Second * 1):
			log.Printf("Dropped message from client: %s\n", string(msg))
		case client.recv <- msg:
		}
	}
}

func writePump(client *Client) {
	for msg := range client.send {
		_, err := client.conn.Write(msg)
		if err != nil {
			log.Printf("Failed to write to connection: %v\n", err)
			return
		}
	}
}

func interceptor(client *Client, hub *Hub) {
	for data := range client.recv {
		log.Printf("Intercepted message from client\n")
		hub.broadcast <- NewMessage(client, data)
	}
}
