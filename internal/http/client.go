package http

import (
	"context"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/owenHochwald/Volt/internal/apperror"
	"github.com/owenHochwald/Volt/internal/logging"
)

const (
	TIMEOUT = 10 * time.Second
)

type Client struct {
	Timeout   time.Duration
	RoundTrip bool

	Transport *http.Transport
	Client    *http.Client
}

// makeCustomRequest builds the outgoing *http.Request and dispatches it
// under ctx - which callers (see Send) must keep alive for as long as
// they intend to read the response body, not just until this function
// returns. c.Client.Do only blocks for the response's headers; on a
// streamed body (the normal case for anything but a tiny response) the
// body is read separately, afterward, by the caller - a context that's
// already been cancelled by then aborts that read with "context
// canceled", even though the request itself succeeded.
func (c *Client) makeCustomRequest(ctx context.Context, req *Request) (*http.Response, error) {
	payload := strings.NewReader(req.Body)
	customReq, err := http.NewRequest(req.Method, req.URL, payload)
	if err != nil {
		return nil, err
	}

	for key, val := range req.Headers {
		customReq.Header.Set(key, val)
	}
	customReq = customReq.WithContext(ctx)

	return c.Client.Do(customReq)
}

func (c *Client) Send(req *Request, result chan<- *Response) {
	var start time.Time

	if c.RoundTrip {
		// time for connections, headers, and body
		start = time.Now()
	}

	// traceStart anchors the DNS/connect/TLS/TTFB breakdown (see Timing)
	// independent of the RoundTrip toggle above - that toggle only
	// changes what the aggregate Duration field measures, not when these
	// finer-grained phases actually happened.
	traceStart := time.Now()
	timing, trace := newTimingTrace(traceStart)

	// This context's cancellation must stay in scope for the whole
	// send - headers *and* body - not just until makeCustomRequest
	// returns (that used to be a real bug: deferring cancel() inside
	// makeCustomRequest cancelled the request's context the moment
	// c.Client.Do got the response headers back, before the body below
	// was read at all. A response small enough to already be fully
	// buffered read fine regardless (nothing left to block on), but
	// anything that actually needed a further blocking read off the
	// socket - any response of meaningful size - failed with "context
	// canceled", looking like a random, response-size-dependent failure
	// with no connection to what was actually wrong).
	ctx := context.Background()
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}
	ctx = httptrace.WithClientTrace(ctx, trace)

	logging.Logf("http: -> %s %s (%d header(s), %d byte body)", req.Method, req.URL, len(req.Headers), len(req.Body))

	res, err := c.makeCustomRequest(ctx, req)
	if err != nil {
		logging.Logf("http: <- %s %s failed: %v", req.Method, req.URL, err)
		failure := apperror.FromNetwork(err)
		result <- &Response{Error: failure.Message, Failure: failure}
		return
	}

	if !c.RoundTrip {
		// time for just first byte
		start = time.Now()
	}

	// close body from earlier call
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		logging.Logf("http: <- %s %s: got status %s but failed reading body: %v", req.Method, req.URL, res.Status, err)
		failure := apperror.FromNetwork(err)
		result <- &Response{Error: failure.Message, Failure: failure}
		return
	}

	duration := time.Since(start)

	logging.Logf("http: <- %s %s -> %s (%d bytes) in %s", req.Method, req.URL, res.Status, len(body), duration)

	result <- &Response{
		Status:     res.Status,
		StatusCode: res.StatusCode,
		Body:       string(body),
		Headers:    res.Header,
		Duration:   duration,
		RoundTrip:  c.RoundTrip,
		Timing:     timing,
	}
}

func (c *Client) ToggleRoundTrip() {
	c.RoundTrip = !c.RoundTrip
}

func InitClient(timeout time.Duration, roundTrip bool) *Client {
	if timeout == 0 {
		timeout = TIMEOUT
	}
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	return &Client{
		Timeout:   timeout,
		RoundTrip: roundTrip,
		Client:    client,
		Transport: tr,
	}
}
