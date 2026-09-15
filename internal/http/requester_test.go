package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJobConfig_BuildWithRequest(t *testing.T) {
	request := NewDefaultRequest()
	request.URL = "http://localhost:8080"
	request.Method = "GET"

	config := &JobConfig{
		Request:       request,
		Concurrency:   5,
		TotalRequests: 50,
		QPS:           0,
		Timeout:       5 * time.Second,
	}

	if config.Request == nil {
		t.Fatal("Request is nil")
	}
	if config.Request.URL == "" {
		t.Fatal("Request URL is empty")
	}
}

func TestJobConfig_RunBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	request := NewDefaultRequest()
	request.URL = server.URL
	request.Method = "GET"

	expectedRequests := 10
	config := &JobConfig{
		Request:       request,
		Concurrency:   2,
		TotalRequests: 10,
		Timeout:       5 * time.Second,
	}

	updates := make(chan *LoadTestStats, 10)

	go func() {
		config.Run(updates)
	}()

	var finalStats *LoadTestStats
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

loop:
	for {
		select {
		case stats, ok := <-updates:
			if !ok {
				break loop
			}
			if stats != nil {
				finalStats = stats
			}
		case <-timer.C:
			t.Fatal("Timeout waiting for load test to finish")
		}
	}

	assert.NotNil(t, finalStats, "Final stats are nil")
	assert.Equal(t, expectedRequests, finalStats.CompletedRequests, "Expected %d requests, got %d", expectedRequests, finalStats.CompletedRequests)
	assert.Equal(t, expectedRequests, finalStats.TotalRequests, "Expected %d requests, got %d", expectedRequests, finalStats.TotalRequests)
	assert.Equal(t, 0, finalStats.FailedRequests, "Expected 0 failed requests, got %d", finalStats.FailedRequests)
	assert.NotEqual(t, 0, finalStats.TotalDuration, "Total duration is 0")
	assert.NotEqual(t, 0, finalStats.StartTime, "Start time is 0")
	assert.NotEqual(t, 0, finalStats.EndTime, "End time is 0")
	assert.NotEqual(t, 0, finalStats.MaxDuration, "Max duration is 0")
	assert.Truef(t, finalStats.MinDuration < finalStats.MaxDuration, "Min duration is greater than max duration (%d > %d)", finalStats.MinDuration, finalStats.MaxDuration)
	assert.NotNil(t, finalStats.Percentiles, "Percentiles are nil")

}

func TestJobConfig_RunHard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	request := NewDefaultRequest()
	request.URL = server.URL
	request.Method = "GET"

	expectedRequests := 100_000
	config := &JobConfig{
		Request:       request,
		Concurrency:   500,
		TotalRequests: expectedRequests,
		Timeout:       5 * time.Second,
	}
	updates := make(chan *LoadTestStats, 10)

	go func() {
		config.Run(updates)
	}()
	var finalStats *LoadTestStats
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

loop:
	for {
		select {
		case stats, ok := <-updates:
			if !ok {
				break loop
			}
			if stats != nil {
				finalStats = stats
			}
		case <-timer.C:
			t.Fatal("Timeout waiting for load test to finish")
		}
	}

	assert.NotNil(t, finalStats, "Final stats are nil")
	assert.Equal(t, expectedRequests, finalStats.CompletedRequests, "Expected %d requests, got %d", expectedRequests, finalStats.CompletedRequests)
	assert.Equal(t, expectedRequests, finalStats.TotalRequests, "Expected %d requests, got %d", expectedRequests, finalStats.TotalRequests)
	assert.Equal(t, 0, finalStats.FailedRequests, "Expected 0 failed requests, got %d", finalStats.FailedRequests)
	assert.NotEqual(t, 0, finalStats.TotalDuration, "Total duration is 0")
	assert.NotEqual(t, 0, finalStats.StartTime, "Start time is 0")
	assert.NotEqual(t, 0, finalStats.EndTime, "End time is 0")
	assert.NotEqual(t, 0, finalStats.MaxDuration, "Max duration is 0")
	assert.Truef(t, finalStats.MinDuration < finalStats.MaxDuration, "Min duration is greater than max duration (%d > %d)", finalStats.MinDuration, finalStats.MaxDuration)
	assert.NotNil(t, finalStats.Percentiles, "Percentiles are nil")
}

func TestJobConfig_RunDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	request := NewDefaultRequest()
	request.URL = server.URL
	request.Method = "GET"

	testDuration := 300 * time.Millisecond
	config := &JobConfig{
		Request:     request,
		Concurrency: 5,
		Duration:    testDuration,
		Timeout:     5 * time.Second,
	}

	updates := make(chan *LoadTestStats, 10)

	start := time.Now()
	go func() {
		config.Run(updates)
	}()

	var finalStats *LoadTestStats
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

loop:
	for {
		select {
		case stats, ok := <-updates:
			if !ok {
				break loop
			}
			if stats != nil {
				finalStats = stats
			}
		case <-timer.C:
			t.Fatal("Timeout waiting for load test to finish")
		}
	}
	elapsed := time.Since(start)

	assert.NotNil(t, finalStats, "Final stats are nil")
	// The run should stop close to the requested duration rather than
	// racing through an estimated fixed request count.
	assert.GreaterOrEqual(t, elapsed, testDuration, "Test stopped before the requested duration elapsed")
	assert.Less(t, elapsed, testDuration+2*time.Second, "Test ran far longer than the requested duration")
	assert.Greater(t, finalStats.CompletedRequests, 0, "Expected at least one completed request")
}

func TestNewLoadTestStats(t *testing.T) {
	stats := NewLoadTestStats(100)

	if stats == nil {
		t.Fatal("NewLoadTestStats returned nil")
	}
	if stats.TotalRequests != 100 {
		t.Errorf("Expected TotalRequests=100, got %d", stats.TotalRequests)
	}
	if stats.Percentiles == nil {
		t.Fatal("Percentiles not initialized")
	}
	if stats.Errors == nil {
		t.Fatal("Errors map not initialized")
	}
}
