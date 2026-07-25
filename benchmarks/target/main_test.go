package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBenchmarkEndpoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		path        string
		status      int
		contentType string
		bodyLength  int
		minDuration time.Duration
	}{
		{name: "empty", path: "/empty", status: http.StatusOK},
		{
			name:        "one kilobyte",
			path:        "/bytes/1024",
			status:      http.StatusOK,
			contentType: "application/octet-stream",
			bodyLength:  1024,
		},
		{
			name:        "json",
			path:        "/json",
			status:      http.StatusOK,
			contentType: "application/json",
			bodyLength:  len("{\"message\":\"volt benchmark\",\"ok\":true}\n"),
		},
		{
			name:        "fixed delay",
			path:        "/delay/10ms",
			status:      http.StatusOK,
			minDuration: 8 * time.Millisecond,
		},
		{
			name:        "server error",
			path:        "/status/500",
			status:      http.StatusInternalServerError,
			contentType: "text/plain; charset=utf-8",
			bodyLength:  len("benchmark error\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			started := time.Now()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()
			newBenchmarkTarget().ServeHTTP(recorder, request)
			response := recorder.Result()
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tt.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, tt.status)
			}
			if got := response.Header.Get("Content-Type"); !strings.HasPrefix(got, tt.contentType) {
				t.Fatalf("content type = %q, want prefix %q", got, tt.contentType)
			}
			if len(body) != tt.bodyLength {
				t.Fatalf("body length = %d, want %d", len(body), tt.bodyLength)
			}
			if elapsed := time.Since(started); elapsed < tt.minDuration {
				t.Fatalf("elapsed = %s, want at least %s", elapsed, tt.minDuration)
			}
		})
	}
}

func TestRequestCountCanBeReadAndReset(t *testing.T) {
	t.Parallel()

	handler := newBenchmarkTarget()

	for range 3 {
		handler.ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/empty", nil),
		)
	}
	if got := readRequestCount(t, handler); got != 3 {
		t.Fatalf("request count = %d, want 3", got)
	}

	reset := httptest.NewRecorder()
	handler.ServeHTTP(
		reset,
		httptest.NewRequest(http.MethodPost, "/__admin/reset", nil),
	)
	if reset.Code != http.StatusNoContent {
		t.Fatalf("reset status = %d, want %d", reset.Code, http.StatusNoContent)
	}
	if got := readRequestCount(t, handler); got != 0 {
		t.Fatalf("request count after reset = %d, want 0", got)
	}
}

func readRequestCount(t *testing.T, handler http.Handler) uint64 {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/__admin/count", nil),
	)

	var result struct {
		Requests uint64 `json:"requests"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result.Requests
}
