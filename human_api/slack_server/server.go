package slack_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
	http.HandleFunc("/interactive", hapiInteractive)
	http.HandleFunc("/health-check", healthCheckHandler)

	// Start the server on port 8080
	log.Println("Starting server on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	return nil
}

func hapiInteractive(w http.ResponseWriter, r *http.Request) {
	// #todo: refactor this function to handle interactive replies.
	// it has to open goroutine and wait for result.
	// after 1 second send async reply 200 in case result is not ready
	//

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

	response := map[string]string{"echo": "reply"}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusBadRequest)
	}
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
	_ = response
	var responseAny map[string]any
	var err error

	domain := strings.Split(text, " ")[0]

	if domain == "wobj" {
		text = text[len("wobj"):]
		text = strings.TrimLeftFunc(text, unicode.IsSpace)

		statusCode, responseAny, err = wobjHandler(text, data["user_name"].(string))
		if err != nil {
			response = map[string]string{"error": fmt.Sprintf("%v", err)}
			statusCode = http.StatusBadRequest
		}
	} else if domain == "help" {
		response = map[string]string{"help": "Show this menu",
			"wobj init": "Init wobject sample for submitting"}
		statusCode = http.StatusOK
	} else {
		statusCode = http.StatusBadRequest
		response = map[string]string{"error": fmt.Sprintf("Unknown domain: %s", domain)}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(responseAny); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusBadRequest)
	}

}

func wobjHandler(text, user_name string) (statusCode int, response map[string]any, err error) {
	command := strings.Split(text, " ")[0]

	if command == "menu" {
		response, err = loadJsonFile("slack_menu.json")
		if err != nil {
			statusCode = http.StatusInternalServerError
			return statusCode, nil, err

		} else {
			statusCode = http.StatusOK
		}

	} else {
		//response = wobjHelp()
		statusCode = http.StatusBadRequest
	}

	return statusCode, response, nil
}

func loadJsonFile(filePath string) (map[string]any, error) {

	jsonData, err := os.ReadFile("/opt/slack_bot_kit/" + filePath)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func wobjHelp() (response map[string]string) {
	response = map[string]string{"init": "Init wobject"}
	return response
}

func AsyncResponse(statusCode int, response map[string]any, url string) error {

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(response)
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return err
	}
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	_ = body
	return nil
}
