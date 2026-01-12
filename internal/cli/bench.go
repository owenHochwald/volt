package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/owenHochwald/Volt/internal/http"
)

// RunBench executes the load test with given configuration
func RunBench(config *BenchConfig) error {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Build Request object
	req := &http.Request{
		Method:  config.Method,
		URL:     config.URL,
		Headers: config.Headers,
		Body:    config.Body,
	}

	// Calculate total requests if duration-based
	totalRequests := config.TotalRequests
	if totalRequests == 0 {
		estimatedReqsPerWorkerPerSec := 1000
		totalRequests = config.Concurrency * int(config.Duration.Seconds()) * estimatedReqsPerWorkerPerSec

		if totalRequests < 1 {
			totalRequests = 1
		}
	}

	jobConfig := &http.JobConfig{
		Request:       req,
		Concurrency:   config.Concurrency,
		TotalRequests: totalRequests,
		Timeout:       config.Timeout,
		QPS:           float64(config.RateLimit),
		StreamUpdates: false,
	}

	updates := make(chan *http.LoadTestStats, 1000)

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go jobConfig.Run(updates)

	var finalStats *http.LoadTestStats
	for {
		select {
		case stats, ok := <-updates:
			if !ok {
				// Channel closed, test complete
				return FormatOutput(finalStats, config)
			}
			finalStats = stats

		case <-sigCh:
			fmt.Fprintln(os.Stderr, "\nTest interrupted by user")
			if finalStats != nil {
				return FormatOutput(finalStats, config)
			}
			return nil
		}
	}
}
