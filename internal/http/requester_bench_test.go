package http

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

// setupServer starts a high-performance fasthttp server on a random port
func setupServer(latency time.Duration) (string, func()) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	handler := func(ctx *fasthttp.RequestCtx) {
		if latency > 0 {
			time.Sleep(latency)
		}
		ctx.SetStatusCode(fasthttp.StatusOK)
	}

	server := &fasthttp.Server{
		Handler:               handler,
		NoDefaultServerHeader: true,
		DisableKeepalive:      false,
	}

	go func() {
		if err := server.Serve(ln); err != nil {
		}
	}()

	return "http://" + ln.Addr().String(), func() {
		ln.Close()
	}
}

// BenchmarkEngine_SingleRequest measures allocations per individual request execution.
//
// Usage:
//
//	go test -bench=BenchmarkEngine_SingleRequest -benchmem ./internal/http
func BenchmarkEngine_SingleRequest(b *testing.B) {
	serverURL, stop := setupServer(0)
	defer stop()

	req := NewDefaultRequest()
	req.URL = serverURL
	req.Method = "GET"

	concurrencyLevels := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				updates := make(chan *LoadTestStats, 100)

				done := make(chan struct{})
				go func() {
					for range updates {
					}
					close(done)
				}()

				config := &JobConfig{
					Request:       req,
					Concurrency:   concurrency,
					TotalRequests: concurrency,
					Timeout:       5 * time.Second,
				}

				config.Run(context.Background(), updates)
				<-done
			}
		})
	}
}

// BenchmarkEngine_Throughput measures maximum request throughput. Must start
// a test server.
//
// Usage:
//
//	go test -bench=BenchmarkEngine_Throughput -benchmem ./internal/http
//	go test -bench=BenchmarkEngine_Throughput -benchtime=10s ./internal/http
func BenchmarkEngine_Throughput(b *testing.B) {
	serverURL, stop := setupServer(0)
	defer stop()

	req := NewDefaultRequest()
	req.URL = serverURL
	req.Method = "GET"

	concurrencyLevels := []int{10, 50, 100, 500}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			b.ResetTimer()

			updates := make(chan *LoadTestStats, 100)
			done := make(chan struct{})

			go func() {
				for range updates {
				}
				close(done)
			}()

			config := &JobConfig{
				Request:       req,
				Concurrency:   concurrency,
				TotalRequests: b.N,
				Timeout:       5 * time.Second,
			}

			config.Run(context.Background(), updates)
			<-done

			b.StopTimer()

			elapsed := b.Elapsed().Seconds()
			rps := float64(b.N) / elapsed
			b.ReportMetric(rps, "req/s")
		})
	}
}

// BenchmarkEngine_Latency tests performance with slow server responses.
func BenchmarkEngine_Latency(b *testing.B) {
	serverURL, stop := setupServer(50 * time.Millisecond)
	defer stop()

	req := NewDefaultRequest()
	req.URL = serverURL
	req.Method = "GET"

	concurrency := 100

	b.ResetTimer()

	updates := make(chan *LoadTestStats, 100)
	done := make(chan struct{})

	go func() {
		for range updates {
		}
		close(done)
	}()

	config := &JobConfig{
		Request:       req,
		Concurrency:   concurrency,
		TotalRequests: b.N,
		Timeout:       10 * time.Second,
	}

	config.Run(context.Background(), updates)
	<-done

	b.StopTimer()

	elapsed := b.Elapsed().Seconds()
	rps := float64(b.N) / elapsed
	b.ReportMetric(rps, "req/s")
}
