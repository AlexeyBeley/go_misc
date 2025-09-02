package slack_server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func mainHandler(w http.ResponseWriter, r *http.Request) {
	input := []byte{}
	intReadBytes, err := r.Body.Read(input)
	_ = intReadBytes
	strRet := string(input)
	if err != nil {

	}

	fmt.Fprintf(w, "Hello, World! You requested: %s, params: %v, paramsStr: %s\n", r.URL.Path, r.Body, strRet)
	log.Printf("Hello, World! You requested: %s, params: %v, paramsStr: %s\n", r.URL.Path, r.Body, strRet)
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

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Example of a struct to be returned as JSON

	fmt.Fprint(w, "OK")
}

func Start() error {
	// Register the handler functions for different paths
	http.HandleFunc("/ticket", mainHandler)
	http.HandleFunc("/health-check", healthCheckHandler)

	// Start the server on port 8080
	log.Println("Starting server on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	return nil
}
