package cli

import (
	"context"
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

	jobConfig := &http.JobConfig{
		Request:          req,
		Concurrency:      config.Concurrency,
		TotalRequests:    config.TotalRequests,
		Duration:         config.Duration,
		Timeout:          config.Timeout,
		QPS:              float64(config.RateLimit),
		DisableKeepAlive: !config.KeepAlive,
		StreamUpdates:    false,
	}

	updates := make(chan *http.LoadTestStats, 1000)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go jobConfig.Run(ctx, updates)

	var finalStats *http.LoadTestStats
	for stats := range updates {
		if stats != nil {
			finalStats = stats
		}
	}
	if ctx.Err() != nil {
		fmt.Fprintln(os.Stderr, "\nTest interrupted by user")
	}
	if finalStats == nil {
		return nil
	}
	return FormatOutput(finalStats, config)
}
