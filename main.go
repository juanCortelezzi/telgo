package main

import (
	"log"
	"net"
)

func handleConnection(conn *net.TCPConn, hub *Hub) {
	client := NewClient(conn)

	go WritePump(client)
	go Interceptor(client, hub)
	ReadPump(client, hub)
}

func main() {
	addr := net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 6969,
		Zone: "",
	}

	log.Println("Starting server on port 6969")

	listener, err := net.ListenTCP("tcp", &addr)
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	hub := NewHub()
	go hub.run()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			log.Printf("Failed to cast connection to TCPConn\n")
			err := conn.Close()
			if err != nil {
				log.Printf("Failed to close connection: %v\n", err)
			}
			continue
		}

		go handleConnection(tcpConn, hub)
	}
}
