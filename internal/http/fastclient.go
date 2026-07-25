package http

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

type HeaderEntry struct {
	Key   []byte
	Value []byte
}

type FastRequest struct {
	Method  []byte
	URL     []byte
	Headers []HeaderEntry
	Body    []byte
}

type FastClient struct {
	client    *fasthttp.Client
	timeout   time.Duration
	keepAlive bool
	conns     connectionTracker
}

func NewFastClient(timeout time.Duration, s *JobConfig) *FastClient {
	f := &FastClient{
		timeout:   timeout,
		keepAlive: !s.DisableKeepAlive,
		client: &fasthttp.Client{
			MaxConnsPerHost:           max(500, s.Concurrency),
			MaxIdleConnDuration:       10 * time.Second,
			ReadTimeout:               s.Timeout,
			WriteTimeout:              s.Timeout,
			MaxConnDuration:           0,
			MaxIdemponentCallAttempts: 1,
		},
	}
	f.client.DialTimeout = f.dialTimeout
	return f
}

func (f *FastClient) dialTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	conn, err := fasthttp.DialTimeout(addr, timeout)
	if err != nil {
		return nil, err
	}
	return f.conns.track(conn)
}

// Cancel interrupts active requests by closing this run's connections.
func (f *FastClient) Cancel() {
	f.conns.cancel()
}

func (f *FastClient) CloseIdleConnections() {
	f.client.CloseIdleConnections()
}

func (f *FastClient) Do(
	fr *FastRequest,
	req *fasthttp.Request,
	res *fasthttp.Response,
) (status int, bytesSent int64, bytesRecv int64, err error) {
	req.Reset()
	res.Reset()

	// zero allocation setters
	req.Header.SetMethodBytes(fr.Method)
	req.SetRequestURIBytes(fr.URL)
	for _, entry := range fr.Headers {

		req.Header.SetBytesKV(entry.Key, entry.Value)
	}
	if fr.Body != nil {
		req.SetBodyRaw(fr.Body)
	}
	if !f.keepAlive {
		req.Header.SetConnectionClose()
	}

	err = f.client.DoTimeout(req, res, f.timeout)
	if err != nil {
		return 0, int64(len(fr.Body)), int64(len(res.Body())), err
	}

	return res.StatusCode(), int64(len(fr.Body)), int64(len(res.Body())), nil
}

type connectionTracker struct {
	mu       sync.Mutex
	conns    map[net.Conn]struct{}
	canceled bool
}

func (t *connectionTracker) track(conn net.Conn) (net.Conn, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.canceled {
		_ = conn.Close()
		return nil, context.Canceled
	}
	if t.conns == nil {
		t.conns = make(map[net.Conn]struct{})
	}
	t.conns[conn] = struct{}{}
	return &trackedConn{Conn: conn, tracker: t}, nil
}

func (t *connectionTracker) remove(conn net.Conn) {
	t.mu.Lock()
	delete(t.conns, conn)
	t.mu.Unlock()
}

func (t *connectionTracker) cancel() {
	t.mu.Lock()
	t.canceled = true
	conns := make([]net.Conn, 0, len(t.conns))
	for conn := range t.conns {
		conns = append(conns, conn)
	}
	clear(t.conns)
	t.mu.Unlock()

	for _, conn := range conns {
		_ = conn.Close()
	}
}

type trackedConn struct {
	net.Conn
	tracker *connectionTracker
	once    sync.Once
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.tracker.remove(c.Conn) })
	return err
}
