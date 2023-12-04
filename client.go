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

var clientIDCounter atomic.Uint64

type Client struct {
	Id   ID
	Conn *net.TCPConn
	Send chan []byte
	Recv chan []byte
}

func NewClient(conn *net.TCPConn) *Client {
	return &Client{
		Id:   clientIDCounter.Add(1),
		Conn: conn,
		Send: make(chan []byte),
		Recv: make(chan []byte),
	}
}

func (client *Client) Close() {
	close(client.Send)
	close(client.Recv)
	client.Conn.Close()
}

func IsValidMessage(msg []byte) bool {
	for _, r := range bytes.Runes(msg) {
		if r <= 0 || r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func ReadPump(client *Client, hub *Hub) {
	defer (func() { hub.unregister <- client })()

	msgBuffer := make([]byte, 32)
	for {
		pointer, err := client.Conn.Read(msgBuffer)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("Failed to read from connection: %v\n", err)
			}
			return
		}

		msg := msgBuffer[:pointer]

		if !IsValidMessage(msg) {
			log.Printf("Invalid message from client: `%s`\n", string(msg))
			continue
		}

		log.Printf("Received message from client: `%s`\n", string(msg))

		select {
		case <-time.After(time.Second * 1):
			log.Printf("Dropped message from client: %s\n", string(msg))
		case client.Recv <- msg:
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
		hub.broadcast <- NewMessage(client, data)
	}
}
