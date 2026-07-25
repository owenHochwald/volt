package http

import (
	"context"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runJob(t *testing.T, ctx context.Context, config *JobConfig) *LoadTestStats {
	t.Helper()
	updates := make(chan *LoadTestStats, 16)
	done := make(chan struct{})
	go func() {
		config.Run(ctx, updates)
		close(done)
	}()

	var final *LoadTestStats
	for stats := range updates {
		final = stats
	}
	<-done
	require.NotNil(t, final)
	return final
}

func requestFor(url, method, body string) *Request {
	request := NewDefaultRequest()
	request.URL = url
	request.Method = method
	request.Body = body
	return request
}

func TestJobConfigCountModeHasExactAccountingAndTransferStats(t *testing.T) {
	const responseBody = "response"
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		_, _ = w.Write([]byte(responseBody))
	}))
	defer server.Close()

	const requests = 10_000
	const requestBody = "ping"
	stats := runJob(t, context.Background(), &JobConfig{
		Request:       requestFor(server.URL, "POST", requestBody),
		Concurrency:   32,
		TotalRequests: requests,
		Timeout:       5 * time.Second,
	})

	assert.Equal(t, requests, stats.CompletedRequests)
	assert.Equal(t, requests, stats.TotalRequests)
	assert.Zero(t, stats.FailedRequests)
	assert.Equal(t, int64(requests*len(requestBody)), stats.BytesSent)
	assert.Equal(t, int64(requests*len(responseBody)), stats.BytesRecv)
	assert.Equal(t, int64(requests), stats.StatusCodes[stdhttp.StatusOK])
	assert.Equal(t, requests, int(stats.Percentiles.digest.Count()))
	assert.Positive(t, stats.TotalDuration)
	assert.Positive(t, stats.MeanDuration())
	assert.Positive(t, stats.MinDuration)
	assert.GreaterOrEqual(t, stats.MaxDuration, stats.MinDuration)
}

func TestJobConfigWithNoPositiveExecutionBoundFinishesWithoutWork(t *testing.T) {
	tests := map[string]JobConfig{
		"zero bounds":     {Concurrency: 4},
		"negative bounds": {Concurrency: 4, Duration: -time.Second, TotalRequests: -1},
	}
	for name, config := range tests {
		t.Run(name, func(t *testing.T) {
			updates := make(chan *LoadTestStats, 1)
			done := make(chan struct{})
			go func() {
				config.Run(context.Background(), updates)
				close(done)
			}()

			select {
			case stats, ok := <-updates:
				require.True(t, ok)
				require.NotNil(t, stats)
				assert.Zero(t, stats.TotalRequests)
				assert.Zero(t, stats.CompletedRequests)
				assert.Zero(t, stats.FailedRequests)
				assert.Zero(t, stats.MinDuration)
				assert.False(t, stats.EndTime.IsZero())
			case <-time.After(time.Second):
				t.Fatal("zero-work job did not terminate")
			}

			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("zero-work job did not return")
			}
			_, ok := <-updates
			assert.False(t, ok, "updates channel should be closed")
		})
	}
}

func TestJobConfigDurationModeRunsForConfiguredDuration(t *testing.T) {
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		time.Sleep(2 * time.Millisecond)
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer server.Close()

	const duration = 150 * time.Millisecond
	stats := runJob(t, context.Background(), &JobConfig{
		Request:     requestFor(server.URL, "GET", ""),
		Concurrency: 4,
		Duration:    duration,
		Timeout:     time.Second,
	})

	elapsed := stats.EndTime.Sub(stats.StartTime)
	assert.GreaterOrEqual(t, elapsed, duration-10*time.Millisecond)
	assert.Less(t, elapsed, duration+250*time.Millisecond)
	assert.Positive(t, stats.CompletedRequests)
	assert.Equal(t, stats.CompletedRequests, stats.TotalRequests)
}

func TestJobConfigRateLimitIsGlobalAcrossWorkers(t *testing.T) {
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer server.Close()

	stats := runJob(t, context.Background(), &JobConfig{
		Request:       requestFor(server.URL, "GET", ""),
		Concurrency:   6,
		TotalRequests: 6,
		QPS:           10,
		Timeout:       time.Second,
	})

	// One immediate start plus five globally spaced 100 ms starts.
	elapsed := stats.EndTime.Sub(stats.StartTime)
	assert.GreaterOrEqual(t, elapsed, 450*time.Millisecond)
	assert.Less(t, elapsed, 2*time.Second)
	assert.Equal(t, 6, stats.CompletedRequests)
}

func TestJobConfigContextCancellationStopsWorkersAndReturnsFinalStats(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel()
	}()

	start := time.Now()
	stats := runJob(t, ctx, &JobConfig{
		Request:       requestFor(server.URL, "GET", ""),
		Concurrency:   20,
		TotalRequests: 1_000_000,
		Timeout:       5 * time.Second,
	})

	assert.Less(t, time.Since(start), time.Second)
	assert.LessOrEqual(t, stats.CompletedRequests, 20)
	assert.GreaterOrEqual(t, stats.CompletedRequests, 1)
	assert.Equal(t, int64(stats.CompletedRequests), stats.Errors["canceled"])
	assert.Equal(t, stats.CompletedRequests, int(stats.Percentiles.digest.Count()))
}

func TestPercentilesUseRequestDistribution(t *testing.T) {
	calculator := newPercentileCalculator()
	for i := 0; i < 90; i++ {
		calculator.add(time.Millisecond)
	}
	for i := 0; i < 10; i++ {
		calculator.add(100 * time.Millisecond)
	}

	assert.Less(t, calculator.Percentile(50), 5*time.Millisecond)
	assert.Greater(t, calculator.Percentile(95), 50*time.Millisecond)
}

func TestPercentilesPreserveSubMillisecondPrecision(t *testing.T) {
	calculator := newPercentileCalculator()
	calculator.add(700 * time.Microsecond)

	assert.Equal(t, 700*time.Microsecond, calculator.Percentile(50))
}

func TestMeanLatencyUsesEveryCompletedRequest(t *testing.T) {
	state := &runState{stats: NewLoadTestStats(2)}
	batch := &requestBatch{
		count:     2,
		total:     400 * time.Microsecond,
		min:       100 * time.Microsecond,
		max:       300 * time.Microsecond,
		latencies: [statsBatchSize]time.Duration{100 * time.Microsecond, 300 * time.Microsecond},
		statuses:  [statsBatchSize]uint16{200, 200},
	}
	state.mergeBatch(batch)

	assert.Equal(t, 200*time.Microsecond, state.stats.MeanDuration())
	assert.Equal(t, 2, state.stats.CompletedRequests)
	assert.Equal(t, 2, int(state.stats.Percentiles.digest.Count()))
}

func TestHTTPFailuresAndStatusCodesAreClassified(t *testing.T) {
	var sequence atomic.Int64
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		switch sequence.Add(1) {
		case 1:
			w.WriteHeader(stdhttp.StatusBadRequest)
		case 2:
			w.WriteHeader(stdhttp.StatusServiceUnavailable)
		default:
			w.WriteHeader(stdhttp.StatusNoContent)
		}
	}))
	defer server.Close()

	stats := runJob(t, context.Background(), &JobConfig{
		Request:       requestFor(server.URL, "GET", ""),
		Concurrency:   1,
		TotalRequests: 3,
		Timeout:       time.Second,
	})

	assert.Equal(t, 2, stats.FailedRequests)
	assert.Equal(t, int64(1), stats.StatusCodes[stdhttp.StatusBadRequest])
	assert.Equal(t, int64(1), stats.StatusCodes[stdhttp.StatusServiceUnavailable])
	assert.Equal(t, int64(1), stats.StatusCodes[stdhttp.StatusNoContent])
	assert.Equal(t, int64(1), stats.Errors["http_4xx"])
	assert.Equal(t, int64(1), stats.Errors["http_5xx"])
}

func TestTransportAndTimeoutErrorsAreClassified(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		url := "http://" + listener.Addr().String()
		require.NoError(t, listener.Close())

		stats := runJob(t, context.Background(), &JobConfig{
			Request:       requestFor(url, "GET", ""),
			Concurrency:   1,
			TotalRequests: 1,
			Timeout:       100 * time.Millisecond,
		})
		assert.Equal(t, 1, stats.CompletedRequests)
		assert.Equal(t, 1, stats.FailedRequests)
		assert.Equal(t, int64(1), stats.Errors["connection_refused"])
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(stdhttp.StatusNoContent)
		}))
		defer server.Close()

		stats := runJob(t, context.Background(), &JobConfig{
			Request:       requestFor(server.URL, "GET", ""),
			Concurrency:   1,
			TotalRequests: 1,
			Timeout:       10 * time.Millisecond,
		})
		assert.Equal(t, 1, stats.FailedRequests)
		assert.Equal(t, int64(1), stats.Errors["timeout"])
	})
}

func TestKeepAliveConfigurationControlsConnectionReuse(t *testing.T) {
	run := func(t *testing.T, keepAlive bool) int64 {
		t.Helper()
		var newConnections atomic.Int64
		server := httptest.NewUnstartedServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			_, _ = w.Write([]byte("ok"))
		}))
		server.Config.ConnState = func(_ net.Conn, state stdhttp.ConnState) {
			if state == stdhttp.StateNew {
				newConnections.Add(1)
			}
		}
		server.Start()
		defer server.Close()

		stats := runJob(t, context.Background(), &JobConfig{
			Request:          requestFor(server.URL, "GET", ""),
			Concurrency:      1,
			TotalRequests:    5,
			Timeout:          time.Second,
			DisableKeepAlive: !keepAlive,
		})
		require.Equal(t, 5, stats.CompletedRequests)
		return newConnections.Load()
	}

	assert.Equal(t, int64(1), run(t, true))
	assert.GreaterOrEqual(t, run(t, false), int64(5))
}

func TestNewLoadTestStats(t *testing.T) {
	stats := NewLoadTestStats(100)
	require.NotNil(t, stats)
	assert.Equal(t, 100, stats.TotalRequests)
	assert.NotNil(t, stats.Percentiles)
	assert.NotNil(t, stats.Errors)
	assert.NotNil(t, stats.StatusCodes)
	assert.Zero(t, stats.MeanDuration())
}
