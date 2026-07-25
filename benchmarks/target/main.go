package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

type benchmarkTarget struct {
	requests atomic.Uint64
}

func newBenchmarkTarget() http.Handler {
	target := &benchmarkTarget{}
	mux := http.NewServeMux()
	oneKilobyte := make([]byte, 1024)
	jsonResponse := []byte("{\"message\":\"volt benchmark\",\"ok\":true}\n")
	errorResponse := []byte("benchmark error\n")

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /__admin/reset", func(w http.ResponseWriter, _ *http.Request) {
		target.requests.Store(0)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /__admin/count", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]uint64{
			"requests": target.requests.Load(),
		})
	})

	mux.HandleFunc("GET /empty", target.count(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
	}))
	mux.HandleFunc("GET /bytes/1024", target.count(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "1024")
		_, _ = w.Write(oneKilobyte)
	}))
	mux.HandleFunc("GET /json", target.count(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonResponse)
	}))
	mux.HandleFunc("GET /delay/10ms", target.count(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
	}))
	mux.HandleFunc("GET /status/500", target.count(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errorResponse)
	}))

	return mux
}

func (t *benchmarkTarget) count(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.requests.Add(1)
		next(w, r)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newBenchmarkTarget(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("benchmark target listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(fmt.Errorf("serve benchmark target: %w", err))
	}
}
