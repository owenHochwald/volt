package http

import (
	"crypto/tls"
	"net/http/httptrace"
	"time"
)

// Timing breaks a single request's round trip down into the phases
// net/http/httptrace can observe - DNS lookup, TCP connect, TLS
// handshake, and time to first response byte - alongside the overall
// Duration already tracked on Response. Populated by Client.Send (see
// client.go); left nil for anything that doesn't go through that path
// (a load test's FastClient, or a response an older session recorded
// into history before this existed).
type Timing struct {
	DNSLookup       time.Duration
	TCPConnection   time.Duration
	TLSHandshake    time.Duration
	TimeToFirstByte time.Duration

	// Reused reports whether the request went out on an existing
	// keep-alive connection - if so, DNSLookup/TCPConnection/TLSHandshake
	// are all zero, not because those phases were instant but because
	// they simply didn't happen for this request (they happened once, on
	// an earlier request that first opened the connection).
	Reused bool
}

// newTimingTrace builds a *Timing alongside the httptrace.ClientTrace that
// fills it in as the request progresses - traceStart anchors
// TimeToFirstByte, measured from when the request started (not from
// whichever phase happens to run first, since a reused connection skips
// straight to writing the request). The trace's hooks all run on
// whatever goroutine net/http's transport calls them from, but never
// concurrently with each other for a single request, so no locking is
// needed for the plain field writes below.
func newTimingTrace(traceStart time.Time) (*Timing, *httptrace.ClientTrace) {
	timing := &Timing{}
	var dnsStart, connectStart, tlsStart time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !dnsStart.IsZero() {
				timing.DNSLookup = time.Since(dnsStart)
			}
		},
		ConnectStart: func(network, addr string) {
			connectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			if !connectStart.IsZero() {
				timing.TCPConnection = time.Since(connectStart)
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tlsStart.IsZero() {
				timing.TLSHandshake = time.Since(tlsStart)
			}
		},
		GotConn: func(info httptrace.GotConnInfo) {
			timing.Reused = info.Reused
		},
		GotFirstResponseByte: func() {
			timing.TimeToFirstByte = time.Since(traceStart)
		},
	}

	return timing, trace
}
