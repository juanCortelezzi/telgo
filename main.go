package main

import (
	"context"
	"log/slog"
	"net"
)

func main() {
	log := slog.Default()
	ctx := NewLoggerContext(context.Background(), log)
	addr := net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 6969,
		Zone: "",
	}

	log.Info("Starting", "port", addr.Port, "ip", addr.IP.String())

	listener, err := net.ListenTCP("tcp", &addr)
	if err != nil {
		log.Error("Failed to start server", "error", err)
	}

	defer listener.Close()

	hub := NewHub()
	go hub.Run(ctx)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Warn("Failed to accept connection", "error", err)
			continue
		}

		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			log.Warn("Failed to cast connection to TCPConn")
			err := conn.Close()
			if err != nil {
				log.Error("Failed to close connection", "error", err)
			}
			continue
		}

		go NewClient(tcpConn).Run(ctx, hub)
	}
}
