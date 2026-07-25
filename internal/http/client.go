package http

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/owenHochwald/Volt/internal/apperror"
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

func (c *Client) makeCustomRequest(req *Request) (*http.Response, error) {
	payload := strings.NewReader(req.Body)
	customReq, err := http.NewRequest(req.Method, req.URL, payload)
	if err != nil {
		return nil, err
	}

	for key, val := range req.Headers {
		customReq.Header.Set(key, val)
	}

	if c.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
		defer cancel()
		customReq = customReq.WithContext(ctx)
	}

	res, err := c.Client.Do(customReq)

	if err != nil {
		return nil, err
	}

	return res, err
}

func (c *Client) Send(req *Request, result chan<- *Response) {
	var start time.Time

	if c.RoundTrip {
		// time for connections, headers, and body
		start = time.Now()
	}

	res, err := c.makeCustomRequest(req)
	if err != nil {
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
		failure := apperror.FromNetwork(err)
		result <- &Response{Error: failure.Message, Failure: failure}
		return
	}

	duration := time.Since(start)

	result <- &Response{
		Status:     res.Status,
		StatusCode: res.StatusCode,
		Body:       string(body),
		Headers:    res.Header,
		Duration:   duration,
		RoundTrip:  c.RoundTrip,
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
