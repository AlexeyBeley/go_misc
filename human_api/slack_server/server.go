package slack_server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func mainHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World! You requested: %s, params: %v\n", r.URL.Path, r.Body)
}

func jsonHandler(w http.ResponseWriter, r *http.Request) {
	// Example of a struct to be returned as JSON
	type Message struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	}

	w.Header().Set("Content-Type", "application/json")
	msg := Message{Message: "Success!", Status: "OK"}

	// Encode the struct to JSON and write to the response writer
	json.NewEncoder(w).Encode(msg)
}

func Start() error {
	// Register the handler functions for different paths
	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/json", jsonHandler)

	// Start the server on port 8080
	log.Println("Starting server on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	return nil
}
