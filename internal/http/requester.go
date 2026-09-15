// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package http

import (
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/influxdata/tdigest"
	"github.com/valyala/fasthttp"
)

// largest size of the buffer for the result channel
const maxResult = 1_000_000

// PercentileCalculator data structure for streaming percentile calculations
type PercentileCalculator struct {
	digest *tdigest.TDigest
}

// LoadTestStats holds aggregated stats about the load test
type LoadTestStats struct {
	StartTime         time.Time
	EndTime           time.Time
	TotalRequests     int
	CompletedRequests int
	FailedRequests    int

	// time tracking
	MinDuration   time.Duration
	MaxDuration   time.Duration
	TotalDuration time.Duration

	// streaming percentile calculations
	Percentiles *PercentileCalculator

	// network stats
	BytesSent int64
	BytesRecv int64

	// error tracking
	Errors map[string]int64 // error code -> count

	// system metrics
	CPUUsage    float64 // percentage
	MemoryUsage uint64  // bytes

	mu sync.RWMutex // protects above fields from concurrent access
}

// workerStats holds per-worker local statistics (no mutex needed)
type workerStats struct {
	requests uint64 // total requests by this worker
	failures uint64 // failed requests

	// Sampled latency tracking (1/256 requests)
	sampledCount uint64
	sampledMin   uint64 // nanoseconds
	sampledMax   uint64 // nanoseconds
	sampledTotal uint64 // sum for average calculation

	errorCodes map[int]uint64
}

// workerStatsMsg is sent from workers to aggregator
type workerStatsMsg struct {
	workerID int
	stats    workerStats
}

type JobConfig struct {
	// Request config
	Request     *Request // base request to send
	FastRequest *FastRequest

	// Load parameters
	Concurrency   int           // number of concurrent requests
	TotalRequests int           // total requests to send (0 when Duration is used instead)
	Duration      time.Duration // if > 0, run for this long instead of a fixed request count
	RateLimit     int           // max requests per second
	Timeout       time.Duration // time per request
	QPS           float64       // rate limit for queries per second
	StreamUpdates bool          // if false, only send final result (for CLI mode)

	// Internal state
	client        *FastClient
	workerStatsCh chan workerStatsMsg
	stopCh        chan struct{}
	start         time.Time
	once          sync.Once
	stats         *LoadTestStats
}

// NewLoadTestStats creates a new LoadTestStats instance
func NewLoadTestStats(totalRequests int) *LoadTestStats {
	return &LoadTestStats{
		StartTime:     time.Now(),
		TotalRequests: totalRequests,
		MinDuration:   time.Duration(math.MaxInt64),
		Percentiles: &PercentileCalculator{
			digest: tdigest.NewWithCompression(100),
		},
		Errors: make(map[string]int64),
	}
}

// Run makes all the requests and streams results
func (s *JobConfig) Run(updates chan<- *LoadTestStats) {
	s.start = time.Now()
	s.stats = NewLoadTestStats(s.TotalRequests)
	s.FastRequest = compileRequest(s.Request)
	s.client = NewFastClient(s.Timeout, s)

	// aggregation channel (buffered for batch flushes)
	s.workerStatsCh = make(chan workerStatsMsg, s.Concurrency*4)

	s.stopCh = make(chan struct{}, s.Concurrency)

	// Duration-based tests: stop all workers once the time budget is up.
	if s.Duration > 0 {
		go func() {
			timer := time.NewTimer(s.Duration)
			defer timer.Stop()
			<-timer.C
			s.once.Do(func() { close(s.stopCh) })
		}()
	}

	// Aggregate in background
	go s.aggregateStats(updates)

	// Run workers
	s.runWorkers()

	// Signal completion
	close(s.workerStatsCh)
}

func compileRequest(req *Request) *FastRequest {
	fastReq := &FastRequest{
		Method:  []byte(req.Method),
		URL:     []byte(req.URL),
		Headers: make([]HeaderEntry, 0, len(req.Headers)),
	}

	for k, v := range req.Headers {
		fastReq.Headers = append(fastReq.Headers, HeaderEntry{
			Key:   []byte(k),
			Value: []byte(v),
		})
	}

	if len(req.Body) > 0 {
		fastReq.Body = []byte(req.Body)
	}

	return fastReq
}

func (p *PercentileCalculator) Percentile(percentile float64) time.Duration {
	ms := p.digest.Quantile(percentile / 100.0)
	return time.Duration(ms) * time.Millisecond
}

func (s *LoadTestStats) GetSnapshot() LoadTestStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	errorsCopy := make(map[string]int64, len(s.Errors))
	for k, v := range s.Errors {
		errorsCopy[k] = v
	}

	return LoadTestStats{
		StartTime:         s.StartTime,
		EndTime:           s.EndTime,
		TotalRequests:     s.TotalRequests,
		CompletedRequests: s.CompletedRequests,
		FailedRequests:    s.FailedRequests,
		MinDuration:       s.MinDuration,
		MaxDuration:       s.MaxDuration,
		TotalDuration:     s.TotalDuration,
		Percentiles:       s.Percentiles, // can share, it's thread safe
		BytesSent:         s.BytesSent,
		BytesRecv:         s.BytesRecv,
		Errors:            errorsCopy,
		CPUUsage:          s.CPUUsage,
		MemoryUsage:       s.MemoryUsage,
	}
}

func (s *JobConfig) runWorker(workerID int, wg *sync.WaitGroup) {
	defer wg.Done()

	// Worker owns these for its ENTIRE lifetime
	req := &fasthttp.Request{}
	res := &fasthttp.Response{}

	// Local stats
	var stats workerStats
	stats.errorCodes = make(map[int]uint64, 8)

	// QPS throttling setup
	var qpsTicker *time.Ticker
	var throttle <-chan time.Time
	if s.QPS > 0 {
		qpsTicker = time.NewTicker(time.Duration(1e6/s.QPS) * time.Microsecond)
		defer qpsTicker.Stop()
		throttle = qpsTicker.C
	}

	// Duration-based tests run until the timer in Run() closes stopCh;
	// count-based tests run a fixed, pre-allocated share of TotalRequests.
	unbounded := s.Duration > 0
	requestsPerWorker := 0
	if !unbounded {
		requestsPerWorker = s.TotalRequests / s.Concurrency
		remainder := s.TotalRequests % s.Concurrency

		// First 'remainder' workers get one extra request
		if workerID < remainder {
			requestsPerWorker++
		}
	}

	for i := 0; unbounded || i < requestsPerWorker; i++ {
		// QPS throttling
		if s.QPS > 0 {
			<-throttle
		}

		// Check for stop signal
		select {
		case <-s.stopCh:
			s.flushWorkerStats(workerID, &stats)
			return
		default:
		}

		// Latency sampling: 1 out of every 256 requests (cheap bitwise check)
		sample := (i & 0xFF) == 0

		var start time.Time
		if sample {
			start = time.Now()
		}

		status, _, err := s.client.Do(s.FastRequest, req, res)

		if sample {
			elapsed := uint64(time.Since(start).Nanoseconds())
			stats.sampledCount++

			if stats.sampledMin == 0 || elapsed < stats.sampledMin {
				stats.sampledMin = elapsed
			}
			if elapsed > stats.sampledMax {
				stats.sampledMax = elapsed
			}
			stats.sampledTotal += elapsed
		}

		stats.requests++
		if err != nil {
			stats.failures++
			if status > 0 {
				stats.errorCodes[status]++
			}
		}

		// Flush every 256 requests
		if (i&0xFF) == 0 && i > 0 {
			s.flushWorkerStats(workerID, &stats)
			for k := range stats.errorCodes {
				delete(stats.errorCodes, k)
			}
			stats.requests = 0
			stats.failures = 0
			stats.sampledCount = 0
			stats.sampledMin = 0
			stats.sampledMax = 0
			stats.sampledTotal = 0
		}
	}

	s.flushWorkerStats(workerID, &stats)
}

// flushWorkerStats sends local worker stats to aggregator (non-blocking)
func (s *JobConfig) flushWorkerStats(workerID int, stats *workerStats) {
	// Drop if aggregator is behind
	select {
	case s.workerStatsCh <- workerStatsMsg{workerID: workerID, stats: *stats}:
	default:
		// Skip this flush if aggregator channel is full
	}
}

func (s *JobConfig) aggregateStats(updates chan<- *LoadTestStats) {
	var ticker *time.Ticker
	var tickerCh <-chan time.Time

	if s.StreamUpdates {
		ticker = time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		tickerCh = ticker.C
	} else {
		// For CLI mode: use a nil channel (will never receive)
		tickerCh = nil
	}

	for {
		select {
		case msg, ok := <-s.workerStatsCh:
			if !ok {
				s.stats.EndTime = time.Now()
				snapshot := s.stats.GetSnapshot()
				updates <- &snapshot
				close(updates)
				return
			}

			s.stats.mu.Lock()
			s.stats.CompletedRequests += int(msg.stats.requests)
			s.stats.FailedRequests += int(msg.stats.failures)

			// Update latency from samples
			if msg.stats.sampledCount > 0 {
				minDur := time.Duration(msg.stats.sampledMin)
				maxDur := time.Duration(msg.stats.sampledMax)

				if s.stats.MinDuration == 0 || minDur < s.stats.MinDuration {
					s.stats.MinDuration = minDur
				}
				if maxDur > s.stats.MaxDuration {
					s.stats.MaxDuration = maxDur
				}

				s.stats.TotalDuration += time.Duration(msg.stats.sampledTotal)

				// Add samples to tdigest (average of worker's samples)
				avgLatency := float64(msg.stats.sampledTotal) / float64(msg.stats.sampledCount)
				s.stats.Percentiles.digest.Add(avgLatency/1e6, float64(msg.stats.sampledCount))
			}

			// Merge error codes
			for code, count := range msg.stats.errorCodes {
				s.stats.Errors[strconv.Itoa(code)] += int64(count)
			}
			s.stats.mu.Unlock()

		case <-tickerCh:
			// When tickerCh is nil, this case is never selected
			snapshot := s.stats.GetSnapshot()
			updates <- &snapshot
		}
	}
}

func (s *JobConfig) runWorkers() {
	var wg sync.WaitGroup

	for i := 0; i < s.Concurrency; i++ {
		wg.Add(1)
		go s.runWorker(i, &wg)
	}

	wg.Wait()
}
