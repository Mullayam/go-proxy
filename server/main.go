package main

import (
	// "bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const (
	listenPort  = ":20347"          // Server listens for user connections
	clientProxy = "127.0.0.1:20348" // Target client address
)

// Simulated user-to-port mapping (Replace with real logic)
var userPortMap = map[string]string{
	"enjoys": "6380", // enjoys.redis.enjoys.in → 127.0.0.1:6380
}

// Extract username from "username.redis.enjoys.in"
func extractUsername(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) > 0 {
		return parts[0] // First part before ".redis.enjoys.in"
	}
	return ""
}
func handleConnection(clientConn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer clientConn.Close()

	buffer := make([]byte, 1024)
	_, err := clientConn.Read(buffer)
	if err != nil {
		log.Println("Error reading from user connection:", err)
		return
	}

	// Extract username from the hostname
	clientIP := clientConn.RemoteAddr().String()
	serverIp := clientConn.LocalAddr().String()
	log.Printf("New connection from %s\n", clientIP)

	host, _, err := net.SplitHostPort("localAddr")
	if err != nil {
		log.Println("Failed to get client host:", err)
		return
	}

	username := extractUsername(host)
	redisPort, exists := userPortMap[username]
	if !exists {
		log.Printf("Unknown user '%s' tried to connect\n", username)
		return
	}
	targetAddr := fmt.Sprintf("127.0.0.1:%s", redisPort)
	log.Printf("Forwarding %s -> %s\n", serverIp, targetAddr)

	// Connect to the backend Redis
	serverConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Println("Error connecting to client:", err)
		return
	}
	defer serverConn.Close()

	// Bidirectional data forwarding
	go io.Copy(serverConn, clientConn)
	io.Copy(clientConn, serverConn)
	// 	Issue: io.Copy() uses a small internal buffer, which is inefficient for Redis protocol streaming. Use io.CopyBuffer() with a larger buffer:
	// 	buf := make([]byte, 4096) // 4KB buffer
	// go io.CopyBuffer(serverConn, clientConn, buf)
	// io.CopyBuffer(clientConn, serverConn, buf)

}

func main() {
	ln, err := net.Listen("tcp", listenPort)
	if err != nil {
		log.Fatal("Failed to start TCP Server as proxy:", err)
	}
	log.Println("TCP Proxy Server listening on port", listenPort)

	var wg sync.WaitGroup
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Goroutine to accept new connections
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println("Connection error:", err)
				continue
			}
			wg.Add(1)
			go handleConnection(conn, &wg)
		}
	}()

	// Wait for termination signal
	<-stop
	log.Println("Shutting down server...")

	ln.Close() // Stop accepting new connections
	wg.Wait()  // Wait for all active connections to complete
	log.Println("Server shutdown completed")
}
