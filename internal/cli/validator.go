package cli

import (
	"errors"
	"strings"
)

// Validate checks BenchConfig for errors
func (c *BenchConfig) Validate() error {
	// URL is required
	if c.URL == "" {
		return errors.New("--url is required")
	}

	// URL must start with http:// or https://
	if !strings.HasPrefix(c.URL, "http://") && !strings.HasPrefix(c.URL, "https://") {
		return errors.New("URL must start with http:// or https://")
	}

	// Method must be valid
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true,
		"DELETE": true, "PATCH": true, "HEAD": true,
	}
	if !validMethods[strings.ToUpper(c.Method)] {
		return errors.New("invalid HTTP method")
	}

	// Concurrency must be positive
	if c.Concurrency <= 0 {
		return errors.New("concurrency must be > 0")
	}

	if c.Duration < 0 {
		return errors.New("duration must be >= 0")
	}
	if c.TotalRequests < 0 {
		return errors.New("total requests must be >= 0")
	}

	// Exactly one positive execution bound is required.
	durationMode := c.Duration > 0
	requestCountMode := c.TotalRequests > 0
	if !durationMode && !requestCountMode {
		return errors.New("must specify either -d (duration) or -n (total requests)")
	}
	if durationMode && requestCountMode {
		return errors.New("-d and -n are mutually exclusive")
	}

	// Timeout must be positive
	if c.Timeout <= 0 {
		return errors.New("timeout must be > 0")
	}

	// Rate limit must be non-negative
	if c.RateLimit < 0 {
		return errors.New("rate limit must be >= 0")
	}

	return nil
}
