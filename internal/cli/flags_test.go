package cli

import (
	"testing"
	"time"
)

func TestParseBenchFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(*testing.T, *BenchConfig)
	}{
		{
			name: "minimal valid config",
			args: []string{"-url", "http://example.com"},
			check: func(t *testing.T, c *BenchConfig) {
				if c.URL != "http://example.com" {
					t.Errorf("URL = %s, want http://example.com", c.URL)
				}
				if c.Method != "GET" {
					t.Errorf("Method = %s, want GET", c.Method)
				}
				if c.Concurrency != 50 {
					t.Errorf("Concurrency = %d, want 50", c.Concurrency)
				}
				if !c.KeepAlive {
					t.Error("KeepAlive should default to true")
				}
			},
		},
		{
			name: "disable keepalive",
			args: []string{"-url", "http://example.com", "-no-keepalive"},
			check: func(t *testing.T, c *BenchConfig) {
				if c.KeepAlive {
					t.Error("KeepAlive should be false")
				}
			},
		},
		{
			name: "custom headers",
			args: []string{
				"-url", "http://example.com",
				"-H", "Content-Type: application/json",
				"-H", "Authorization: Bearer token",
			},
			check: func(t *testing.T, c *BenchConfig) {
				if len(c.Headers) != 2 {
					t.Errorf("got %d headers, want 2", len(c.Headers))
				}
				if c.Headers["Content-Type"] != "application/json" {
					t.Error("Content-Type header not set correctly")
				}
			},
		},
		{
			name: "duration parsing",
			args: []string{"-url", "http://example.com", "-d", "5m"},
			check: func(t *testing.T, c *BenchConfig) {
				if c.Duration != 5*time.Minute {
					t.Errorf("Duration = %v, want 5m", c.Duration)
				}
			},
		},
		{
			name: "all flags",
			args: []string{
				"-url", "http://example.com",
				"-m", "POST",
				"-b", `{"key":"value"}`,
				"-c", "100",
				"-d", "30s",
				"-t", "60s",
				"-rate", "1000",
				"-json",
				"-q",
				"-o", "results.json",
			},
			check: func(t *testing.T, c *BenchConfig) {
				if c.Method != "POST" {
					t.Errorf("Method = %s, want POST", c.Method)
				}
				if c.Body != `{"key":"value"}` {
					t.Errorf("Body = %s, want {\"key\":\"value\"}", c.Body)
				}
				if c.Concurrency != 100 {
					t.Errorf("Concurrency = %d, want 100", c.Concurrency)
				}
				if c.Duration != 30*time.Second {
					t.Errorf("Duration = %v, want 30s", c.Duration)
				}
				if c.Timeout != 60*time.Second {
					t.Errorf("Timeout = %v, want 60s", c.Timeout)
				}
				if c.RateLimit != 1000 {
					t.Errorf("RateLimit = %d, want 1000", c.RateLimit)
				}
				if !c.JSON {
					t.Error("JSON should be true")
				}
				if !c.Quiet {
					t.Error("Quiet should be true")
				}
				if c.Output != "results.json" {
					t.Errorf("Output = %s, want results.json", c.Output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseBenchFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBenchFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, config)
			}
		})
	}
}
