package apperror

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
)

func TestNetworkErrorsHaveFriendlyMessagesAndRecoveryHints(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode Code
		message  string
		hint     string
	}{
		{
			name:     "timeout",
			err:      context.DeadlineExceeded,
			wantCode: Timeout,
			message:  "timed out",
			hint:     "increase the timeout",
		},
		{
			name:     "DNS lookup",
			err:      &net.DNSError{Name: "volt.invalid", IsNotFound: true},
			wantCode: DNS,
			message:  "find the host",
			hint:     "Check the URL",
		},
		{
			name:     "connection refused",
			err:      errors.New("dial tcp 127.0.0.1:8080: connect: connection refused"),
			wantCode: ConnectionRefused,
			message:  "connect to the server",
			hint:     "server is running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromNetwork(tt.err)
			if got.Category != Network || got.Code != tt.wantCode {
				t.Fatalf("error = (%s, %s), want (%s, %s)", got.Category, got.Code, Network, tt.wantCode)
			}
			if !containsFold(got.Message, tt.message) {
				t.Fatalf("message = %q, want it to contain %q", got.Message, tt.message)
			}
			if !containsFold(got.Hint, tt.hint) {
				t.Fatalf("hint = %q, want it to contain %q", got.Hint, tt.hint)
			}
		})
	}
}

func containsFold(value, substring string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(substring))
}

func TestValidationAndStorageErrorsAreCategorized(t *testing.T) {
	invalidURL := InvalidURL("example.com")
	if invalidURL.Category != Validation || invalidURL.Code != InvalidURLCode {
		t.Fatalf("invalid URL = (%s, %s)", invalidURL.Category, invalidURL.Code)
	}
	if !containsFold(invalidURL.Hint, "https://example.com") {
		t.Fatalf("invalid URL hint = %q", invalidURL.Hint)
	}

	locked := FromStorage(errors.New("database locked"))
	if locked.Category != Storage || locked.Code != DatabaseLocked {
		t.Fatalf("locked database = (%s, %s)", locked.Category, locked.Code)
	}
	if !containsFold(locked.Hint, "try again") {
		t.Fatalf("storage hint = %q", locked.Hint)
	}
}

func TestLoadTestFailureSummaryPrioritizesRecoverableNetworkFailures(t *testing.T) {
	got := LoadTestFailure(5, map[string]int64{
		string(Timeout): 3,
		"http_5xx":      2,
	})

	if got.Category != Network || got.Code != Timeout {
		t.Fatalf("error = (%s, %s), want (%s, %s)", got.Category, got.Code, Network, Timeout)
	}
	if !containsFold(got.Message, "3 timed out") {
		t.Fatalf("message = %q", got.Message)
	}
	if !containsFold(got.Hint, "increase the timeout") {
		t.Fatalf("hint = %q", got.Hint)
	}
}
