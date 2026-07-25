// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package http

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/influxdata/tdigest"
	"github.com/owenHochwald/Volt/internal/apperror"
	"github.com/valyala/fasthttp"
)

const statsBatchSize = 256

// PercentileCalculator calculates quantiles from every observed request
// latency. Values are stored in nanoseconds so sub-millisecond observations
// retain their precision.
type PercentileCalculator struct {
	mu     sync.Mutex
	digest *tdigest.TDigest
}

func newPercentileCalculator() *PercentileCalculator {
	return &PercentileCalculator{digest: tdigest.NewWithCompression(100)}
}

func (p *PercentileCalculator) add(duration time.Duration) {
	p.mu.Lock()
	p.digest.Add(float64(duration.Nanoseconds()), 1)
	p.mu.Unlock()
}

func (p *PercentileCalculator) addBatch(durations []time.Duration) {
	p.mu.Lock()
	for _, duration := range durations {
		p.digest.Add(float64(duration.Nanoseconds()), 1)
	}
	p.mu.Unlock()
}

func (p *PercentileCalculator) clone() *PercentileCalculator {
	clone := newPercentileCalculator()
	p.mu.Lock()
	clone.digest.AddCentroidList(p.digest.Centroids())
	p.mu.Unlock()
	return clone
}

// Percentile returns the requested percentile while preserving nanosecond
// precision.
func (p *PercentileCalculator) Percentile(percentile float64) time.Duration {
	if p == nil {
		return 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.digest.Count() == 0 {
		return 0
	}

	ns := p.digest.Quantile(percentile / 100.0)
	return time.Duration(math.Round(ns))
}

// LoadTestStats holds aggregated stats about the load test.
type LoadTestStats struct {
	StartTime         time.Time
	EndTime           time.Time
	TotalRequests     int
	CompletedRequests int
	FailedRequests    int

	MinDuration   time.Duration
	MaxDuration   time.Duration
	TotalDuration time.Duration

	Percentiles *PercentileCalculator

	// BytesSent and BytesRecv are application payload bytes. They intentionally
	// exclude transport framing and HTTP headers.
	BytesSent int64
	BytesRecv int64

	// Errors contains failure classes such as http_4xx, http_5xx, timeout, and
	// transport. StatusCodes contains every received HTTP status.
	Errors      map[string]int64
	StatusCodes map[int]int64

	CPUUsage    float64
	MemoryUsage uint64

	mu sync.RWMutex
}

// MeanDuration returns the mean latency across all completed attempts.
func (s *LoadTestStats) MeanDuration() time.Duration {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.CompletedRequests == 0 {
		return 0
	}
	return time.Duration(int64(s.TotalDuration) / int64(s.CompletedRequests))
}

// requestBatch is pooled and passed by pointer so the hot path does not
// allocate for every request or copy latency arrays through the channel.
type requestBatch struct {
	count                 int
	failures              int
	timeoutErrs           int
	dnsErrs               int
	connectionRefusedErrs int
	transportErrs         int
	canceledErrs          int
	bytesSent             int64
	bytesRecv             int64
	total                 time.Duration
	min                   time.Duration
	max                   time.Duration
	latencies             [statsBatchSize]time.Duration
	statuses              [statsBatchSize]uint16
}

func (b *requestBatch) reset() {
	*b = requestBatch{}
}

type JobConfig struct {
	Request     *Request
	FastRequest *FastRequest

	Concurrency      int
	TotalRequests    int
	Duration         time.Duration
	RateLimit        int
	Timeout          time.Duration
	QPS              float64
	DisableKeepAlive bool
	StreamUpdates    bool
}

type runState struct {
	config  *JobConfig
	client  *FastClient
	request *FastRequest
	stats   *LoadTestStats
	batches chan *requestBatch
	pool    sync.Pool
	next    atomic.Uint64
	limiter *globalRateLimiter
}

// globalRateLimiter assigns request start times from one run-wide schedule
// shared by every worker. It deliberately allows an initial request
// immediately, followed by evenly spaced starts (burst size one).
type globalRateLimiter struct {
	mu       sync.Mutex
	next     time.Time
	interval time.Duration
}

func newGlobalRateLimiter(qps float64, start time.Time) *globalRateLimiter {
	if qps <= 0 {
		return nil
	}
	interval := time.Duration(float64(time.Second) / qps)
	if interval < time.Nanosecond {
		interval = time.Nanosecond
	}
	return &globalRateLimiter{next: start, interval: interval}
}

func (l *globalRateLimiter) wait(ctx context.Context) bool {
	if l == nil {
		return true
	}

	l.mu.Lock()
	now := time.Now()
	scheduled := l.next
	if scheduled.Before(now) {
		scheduled = now
	}
	l.next = scheduled.Add(l.interval)
	l.mu.Unlock()

	delay := time.Until(scheduled)
	if delay <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return ctx.Err() == nil
	case <-ctx.Done():
		return false
	}
}

// NewLoadTestStats creates a new LoadTestStats instance.
func NewLoadTestStats(totalRequests int) *LoadTestStats {
	return &LoadTestStats{
		StartTime:     time.Now(),
		TotalRequests: totalRequests,
		MinDuration:   time.Duration(math.MaxInt64),
		Percentiles:   newPercentileCalculator(),
		Errors:        make(map[string]int64),
		StatusCodes:   make(map[int]int64),
	}
}

// Run executes the job until its request count is reached, its configured
// duration expires, or ctx is cancelled. The final exact snapshot is always
// delivered before updates is closed.
func (s *JobConfig) Run(ctx context.Context, updates chan<- *LoadTestStats) {
	if ctx == nil {
		ctx = context.Background()
	}

	start := time.Now()
	if s.Duration <= 0 && s.TotalRequests <= 0 {
		stats := NewLoadTestStats(0)
		stats.StartTime = start
		stats.EndTime = time.Now()
		stats.MinDuration = 0
		snapshot := stats.GetSnapshot()
		updates <- &snapshot
		close(updates)
		return
	}

	runCtx := ctx
	cancel := func() {}
	if s.Duration > 0 {
		runCtx, cancel = context.WithDeadline(ctx, start.Add(s.Duration))
	}
	defer cancel()

	concurrency := s.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	stats := NewLoadTestStats(s.TotalRequests)
	stats.StartTime = start
	state := &runState{
		config:  s,
		client:  NewFastClient(s.Timeout, s),
		request: compileRequest(s.Request),
		stats:   stats,
		batches: make(chan *requestBatch, concurrency*4),
		limiter: newGlobalRateLimiter(s.QPS, start),
	}
	state.pool.New = func() any { return new(requestBatch) }

	aggregateDone := make(chan struct{})
	go state.aggregateStats(updates, aggregateDone)

	cancelWatcherDone := make(chan struct{})
	go func() {
		select {
		case <-runCtx.Done():
			state.client.Cancel()
		case <-cancelWatcherDone:
		}
	}()

	var workers sync.WaitGroup
	workers.Add(concurrency)
	for workerID := 0; workerID < concurrency; workerID++ {
		go state.runWorker(runCtx, &workers)
	}
	workers.Wait()
	close(cancelWatcherDone)
	state.client.CloseIdleConnections()
	close(state.batches)
	<-aggregateDone
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

// GetSnapshot returns an immutable copy suitable for another goroutine.
func (s *LoadTestStats) GetSnapshot() LoadTestStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	errorsCopy := make(map[string]int64, len(s.Errors))
	for k, v := range s.Errors {
		errorsCopy[k] = v
	}
	statusCopy := make(map[int]int64, len(s.StatusCodes))
	for k, v := range s.StatusCodes {
		statusCopy[k] = v
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
		Percentiles:       s.Percentiles.clone(),
		BytesSent:         s.BytesSent,
		BytesRecv:         s.BytesRecv,
		Errors:            errorsCopy,
		StatusCodes:       statusCopy,
		CPUUsage:          s.CPUUsage,
		MemoryUsage:       s.MemoryUsage,
	}
}

func (r *runState) runWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	req := &fasthttp.Request{}
	res := &fasthttp.Response{}
	batch := r.acquireBatch()

	for {
		if ctx.Err() != nil {
			break
		}
		if r.config.TotalRequests > 0 {
			requestNumber := r.next.Add(1)
			if requestNumber > uint64(r.config.TotalRequests) {
				break
			}
		}
		if !r.limiter.wait(ctx) {
			break
		}

		start := time.Now()
		status, bytesSent, bytesRecv, err := r.client.Do(r.request, req, res)
		elapsed := time.Since(start)

		i := batch.count
		batch.latencies[i] = elapsed
		if status > 0 {
			batch.statuses[i] = uint16(status)
		}
		batch.count++
		batch.total += elapsed
		batch.bytesSent += bytesSent
		batch.bytesRecv += bytesRecv
		if batch.min == 0 || elapsed < batch.min {
			batch.min = elapsed
		}
		if elapsed > batch.max {
			batch.max = elapsed
		}

		if err != nil {
			batch.failures++
			if ctx.Err() != nil {
				batch.canceledErrs++
			} else {
				switch apperror.FromNetwork(err).Code {
				case apperror.Timeout:
					batch.timeoutErrs++
				case apperror.DNS:
					batch.dnsErrs++
				case apperror.ConnectionRefused:
					batch.connectionRefusedErrs++
				default:
					batch.transportErrs++
				}
			}
		} else if status >= 400 {
			batch.failures++
		}

		if batch.count == statsBatchSize {
			r.batches <- batch
			batch = r.acquireBatch()
		}
	}

	if batch.count > 0 {
		r.batches <- batch
	} else {
		r.releaseBatch(batch)
	}
}

func (r *runState) acquireBatch() *requestBatch {
	batch := r.pool.Get().(*requestBatch)
	batch.reset()
	return batch
}

func (r *runState) releaseBatch(batch *requestBatch) {
	batch.reset()
	r.pool.Put(batch)
}

func (r *runState) aggregateStats(updates chan<- *LoadTestStats, done chan<- struct{}) {
	defer close(done)
	defer close(updates)

	var ticker *time.Ticker
	var tickerCh <-chan time.Time
	if r.config.StreamUpdates {
		ticker = time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		tickerCh = ticker.C
	}

	for {
		select {
		case batch, ok := <-r.batches:
			if !ok {
				r.stats.mu.Lock()
				r.stats.EndTime = time.Now()
				if r.stats.TotalRequests == 0 {
					r.stats.TotalRequests = r.stats.CompletedRequests
				}
				if r.stats.CompletedRequests == 0 {
					r.stats.MinDuration = 0
				}
				r.stats.mu.Unlock()
				snapshot := r.stats.GetSnapshot()
				updates <- &snapshot
				return
			}
			r.mergeBatch(batch)
			r.releaseBatch(batch)

		case <-tickerCh:
			snapshot := r.stats.GetSnapshot()
			// Progress snapshots are advisory. Do not let a slow terminal
			// consumer stall the request engine.
			select {
			case updates <- &snapshot:
			default:
			}
		}
	}
}

func (r *runState) mergeBatch(batch *requestBatch) {
	r.stats.mu.Lock()
	defer r.stats.mu.Unlock()

	r.stats.CompletedRequests += batch.count
	r.stats.FailedRequests += batch.failures
	r.stats.TotalDuration += batch.total
	r.stats.BytesSent += batch.bytesSent
	r.stats.BytesRecv += batch.bytesRecv
	if r.stats.MinDuration == time.Duration(math.MaxInt64) || batch.min < r.stats.MinDuration {
		r.stats.MinDuration = batch.min
	}
	if batch.max > r.stats.MaxDuration {
		r.stats.MaxDuration = batch.max
	}

	r.stats.Percentiles.addBatch(batch.latencies[:batch.count])
	for i := 0; i < batch.count; i++ {
		status := int(batch.statuses[i])
		if status > 0 {
			r.stats.StatusCodes[status]++
		}
		switch {
		case status >= 400 && status < 500:
			r.stats.Errors["http_4xx"]++
		case status >= 500:
			r.stats.Errors["http_5xx"]++
		}
	}

	if batch.timeoutErrs > 0 {
		r.stats.Errors[string(apperror.Timeout)] += int64(batch.timeoutErrs)
	}
	if batch.dnsErrs > 0 {
		r.stats.Errors[string(apperror.DNS)] += int64(batch.dnsErrs)
	}
	if batch.connectionRefusedErrs > 0 {
		r.stats.Errors[string(apperror.ConnectionRefused)] += int64(batch.connectionRefusedErrs)
	}
	if batch.transportErrs > 0 {
		r.stats.Errors[string(apperror.Transport)] += int64(batch.transportErrs)
	}
	if batch.canceledErrs > 0 {
		r.stats.Errors["canceled"] += int64(batch.canceledErrs)
	}
}
