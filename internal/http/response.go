package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/owenHochwald/Volt/internal/apperror"
)

type ResultMsg struct {
	Response *Response
}

func (r *ResultMsg) String() string {
	if r == nil || r.Response == nil {
		return "empty response"
	}
	if r.Response.Error != "" {
		// TODO: add styles
		return r.Response.Error
	}
	return fmt.Sprintf("%d that took: %d ms", r.Response.StatusCode, r.Response.Duration)
}

func (r *Response) ParseContentType() string {
	return r.Headers.Get("Content-Type")
}

type Response struct {
	StatusCode int             `json:"status_code"`
	Status     string          `json:"status,omitempty"`
	Headers    http.Header     `json:"headers,omitempty"`
	Body       string          `json:"body,omitempty"`
	Duration   time.Duration   `json:"duration,omitempty"`
	Error      string          `json:"error,omitempty"`
	RoundTrip  bool            `json:"round_trip,omitempty"`
	Failure    *apperror.Error `json:"-"`
}
