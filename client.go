package main

import (
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

type ID = uint64

var ClientIDCounter atomic.Uint64

type Client struct {
	id   ID
	conn *net.TCPConn
	send chan []byte
	recv chan []byte
}

func readPump(client *Client) {
	msgBuffer := make([]byte, 32)
	for {
		pointer, err := client.conn.Read(msgBuffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("Failed to read from connection: %v\n", err)
			}
			return
		}

		msg := strings.TrimSpace(string(msgBuffer[:pointer]))

		log.Printf("Received message from client: `%s`\n", msg)

		select {
		case <-time.After(time.Second * 1):
			log.Printf("Dropped message from client: %s\n", msg)
		case client.recv <- []byte(msgBuffer[:pointer]):
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
	for msg := range client.recv {
		log.Printf("Intercepted message from client\n")
		hub.broadcast <- msg
	}
}
