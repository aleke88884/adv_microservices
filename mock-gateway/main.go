package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type notifyRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
}

type notifyResponse struct {
	Status string `json:"status"`
}

type requestLogLine struct {
	Time           string `json:"time"`
	Method         string `json:"method"`
	Path           string `json:"path"`
	IdempotencyKey string `json:"idempotency_key"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	StatusCode     int    `json:"status_code"`
}

// seenKeys tracks idempotency keys seen since gateway started.
var (
	seenKeys   = map[string]struct{}{}
	seenKeysMu sync.Mutex
)

func main() {
	port := os.Getenv("GATEWAY_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/notify", handleNotify)

	log.SetFlags(0)
	log.Printf(`{"time":%q,"level":"info","msg":"mock-gateway starting","port":%q}`,
		time.Now().UTC().Format(time.RFC3339), port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf(`{"time":%q,"level":"error","msg":"server error","error":%q}`,
			time.Now().UTC().Format(time.RFC3339), err.Error())
	}
}

func handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Simulate 20% transient failure.
	if rand.Float64() < 0.20 {
		logRequest(r, req, http.StatusServiceUnavailable)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintln(w, `{"status":"error","message":"simulated transient failure"}`)
		return
	}

	seenKeysMu.Lock()
	_, isDuplicate := seenKeys[req.IdempotencyKey]
	if !isDuplicate {
		seenKeys[req.IdempotencyKey] = struct{}{}
	}
	seenKeysMu.Unlock()

	respStatus := "accepted"
	if isDuplicate {
		respStatus = "duplicate"
	}

	logRequest(r, req, http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	resp := notifyResponse{Status: respStatus}
	json.NewEncoder(w).Encode(resp)
}

func logRequest(r *http.Request, req notifyRequest, statusCode int) {
	entry := requestLogLine{
		Time:           time.Now().UTC().Format(time.RFC3339),
		Method:         r.Method,
		Path:           r.URL.Path,
		IdempotencyKey: req.IdempotencyKey,
		Channel:        req.Channel,
		Recipient:      req.Recipient,
		StatusCode:     statusCode,
	}
	b, _ := json.Marshal(entry)
	log.SetFlags(0)
	log.Println(string(b))
}
