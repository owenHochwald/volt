package cli

import (
	"testing"
	"time"
)

func TestBenchConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *BenchConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with duration",
			config: &BenchConfig{
				URL:         "http://example.com",
				Method:      "GET",
				Concurrency: 10,
				Duration:    10 * time.Second,
				Timeout:     30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid config with total requests",
			config: &BenchConfig{
				URL:           "https://example.com",
				Method:        "POST",
				Concurrency:   100,
				TotalRequests: 1000,
				Timeout:       30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			config: &BenchConfig{
				Method:      "GET",
				Concurrency: 10,
				Duration:    10 * time.Second,
				Timeout:     30 * time.Second,
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "invalid URL scheme",
			config: &BenchConfig{
				URL:         "ftp://example.com",
				Method:      "GET",
				Concurrency: 10,
				Duration:    10 * time.Second,
				Timeout:     30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid method",
			config: &BenchConfig{
				URL:         "http://example.com",
				Method:      "INVALID",
				Concurrency: 10,
				Duration:    10 * time.Second,
				Timeout:     30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero concurrency",
			config: &BenchConfig{
				URL:         "http://example.com",
				Method:      "GET",
				Concurrency: 0,
				Duration:    10 * time.Second,
				Timeout:     30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative concurrency",
			config: &BenchConfig{
				URL:         "http://example.com",
				Method:      "GET",
				Concurrency: -5,
				Duration:    10 * time.Second,
				Timeout:     30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "both duration and total requests",
			config: &BenchConfig{
				URL:           "http://example.com",
				Method:        "GET",
				Concurrency:   10,
				Duration:      10 * time.Second,
				TotalRequests: 1000,
				Timeout:       30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "neither duration nor total requests",
			config: &BenchConfig{
				URL:         "http://example.com",
				Method:      "GET",
				Concurrency: 10,
				Timeout:     30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative duration",
			config: &BenchConfig{
				URL:         "http://example.com",
				Method:      "GET",
				Concurrency: 10,
				Duration:    -time.Second,
				Timeout:     30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative total requests",
			config: &BenchConfig{
				URL:           "http://example.com",
				Method:        "GET",
				Concurrency:   10,
				Duration:      10 * time.Second,
				TotalRequests: -1,
				Timeout:       30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: &BenchConfig{
				URL:         "http://example.com",
				Method:      "GET",
				Concurrency: 10,
				Duration:    10 * time.Second,
				Timeout:     0,
			},
			wantErr: true,
		},
		{
			name: "negative rate limit",
			config: &BenchConfig{
				URL:         "http://example.com",
				Method:      "GET",
				Concurrency: 10,
				Duration:    10 * time.Second,
				Timeout:     30 * time.Second,
				RateLimit:   -100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
