package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	localAddr  = "127.0.0.1:6380" // Redis
	serverPort = ":20348"         // Port to listen on
)

func handleConnection(clientConn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	defer clientConn.Close()

	// Connect to the target local service (Redis)
	localConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		log.Println("Error connecting to local service:", err)
		return
	}
	defer localConn.Close()

	// Start bidirectional data copying
	go io.Copy(localConn, clientConn) // Client -> Local
	io.Copy(clientConn, localConn)    // Local -> Client
}

func main() {
	ln, err := net.Listen("tcp", serverPort)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
	log.Println("Server listening on", serverPort)

	var wg sync.WaitGroup
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Goroutine to accept new connections continuously
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println("Connection accept error:", err)
				continue
			}

			// Increment WaitGroup and handle connection concurrently
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
