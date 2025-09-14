package slack_server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"
)

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Example of a struct to be returned as JSON
	fmt.Fprint(w, "OK")
}

func Start() error {
	// Register the handler functions for different paths
	http.HandleFunc("/hapi", hapiMain)
	http.HandleFunc("/health-check", healthCheckHandler)

	// Start the server on port 8080
	log.Println("Starting server on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	return nil
}

func hapiMain(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	// Log the received data

	if !strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
		http.Error(w, "Bad Request. Expected Content-Type 'application/x-www-form-urlencoded', received: "+contentType, http.StatusBadRequest)
		log.Println("Received POST request with no data.")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request: Failed to parse form data", http.StatusBadRequest)
		log.Printf("Error parsing form: %v", err)
		return
	}

	data := make(map[string]any)
	for key, values := range r.Form {
		data[key] = values[0] // Take the first value for each key
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	log.Printf("Received hapi command at %s (Content-Type: %s): %+v", timestamp, contentType, data)

	if len(data) == 0 {
		http.Error(w, "Bad request: Empty data", http.StatusBadRequest)
		log.Printf("Bad request: Empty data")
		return
	}

	text := data["text"].(string)
	text = strings.TrimLeftFunc(text, unicode.IsSpace)

	var statusCode int
	var response map[string]string
	var err error

	domain := strings.Split(text, " ")[0]

	if domain == "wobj" {
		text = text[len("wobj"):]
		text = strings.TrimLeftFunc(text, unicode.IsSpace)

		statusCode, response, err = wobjHandler(text, data["user_name"].(string))
		if err != nil {
			response = map[string]string{"error": fmt.Sprintf("%v", err)}
			statusCode = http.StatusBadRequest
		}
	} else {
		statusCode = http.StatusBadRequest
		response = map[string]string{"error": fmt.Sprintf("Unknown domain: %s", domain)}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusBadRequest)
	}

}

func wobjHandler(text, user_name string) (statusCode int, response map[string]string, err error) {
	command := strings.Split(text, " ")[0]

	if command == "init" {
		response, err = getWobjTemplate(user_name)
		if err != nil {
			statusCode = http.StatusInternalServerError
			return statusCode, nil, err

		} else {
			statusCode = http.StatusOK

		}

	} else {
		response = wobjHelp()
		statusCode = http.StatusBadRequest
	}
	return statusCode, response, nil
}

func getWobjTemplate(requester string) (map[string]string, error) {
	response := map[string]string{
		"Type":        "type",
		"Owner":       requester,
		"Team":        "team name",
		"Title":       "7 > words > 3",
		"Description": "words > 9",
	}
	return response, nil
}

func wobjHelp() (response map[string]string) {
	response = map[string]string{"init": "Init wobject"}
	return response
}
