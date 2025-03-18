package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

const (
	serverPort = ":20348" // Port to listen on
)

func EnvArguments() string {
	// Define port flag
	port := flag.Int("port", 0, "Port number to use")
	flag.IntVar(port, "p", 0, "Port number to use (shorthand)")

	// Define help flag
	help := flag.Bool("help", false, "Show help")
	flag.BoolVar(help, "h", false, "Show help (shorthand)")

	// Parse flags
	flag.Parse()

	// Show help if -h or --help is used
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// Check if port is provided
	if *port == 0 {
		log.Println("Error: --port or -p is required")
		flag.Usage()
		os.Exit(1)
	}

	return strconv.Itoa(*port) // Convert int to string before returning
}
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
	// port := EnvArguments()

	// localAddr := fmt.Printf("127.0.0.1:%s", port)

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
