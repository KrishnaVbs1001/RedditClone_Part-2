package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	engine := NewRedditEngine()

	// Create and start the API server
	server := NewAPIServer(engine)
	go func() {
		log.Printf("Starting API server on :8080...")
		if err := server.Start(":8080"); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(time.Second)

	log.Println("Starting simulation...")
	go RunSimulation("http://localhost:8080", 10) // Simulate 10 users

	fmt.Println("Server and simulation running. Press Ctrl+C to stop...")

	select {}
}
