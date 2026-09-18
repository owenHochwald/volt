// Package apperror defines the small, user-facing error vocabulary used by Volt.
// It deliberately keeps implementation details out of messages shown in the UI.
package apperror

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

type Category string

const (
	Network     Category = "network"
	Validation  Category = "validation"
	Storage     Category = "storage"
	Application Category = "application"
)

type Code string

const (
	Timeout           Code = "timeout"
	DNS               Code = "dns"
	ConnectionRefused Code = "connection_refused"
	Transport         Code = "transport"
	HTTPClientError   Code = "http_4xx"
	HTTPServerError   Code = "http_5xx"

	InvalidURLCode Code = "invalid_url"
	InvalidHeaders Code = "invalid_headers"
	InvalidJSON    Code = "invalid_json"
	InvalidConfig  Code = "invalid_config"

	DatabaseLocked Code = "database_locked"
	StorageIO      Code = "storage_io"

	Unexpected Code = "unexpected"
)

// Error is safe to display to a user. Message explains what happened, and Hint
// is a compact, actionable next step for the notification area.
type Error struct {
	Category Category
	Code     Code
	Message  string
	Hint     string

	cause error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func FromNetwork(err error) *Error {
	if typed, ok := asAppError(err); ok {
		return typed
	}

	lower := strings.ToLower(errorText(err))
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded") {
		return newError(Network, Timeout, "The request timed out.", "Try again or increase the timeout.", err)
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) || strings.Contains(lower, "no such host") || strings.Contains(lower, "server misbehaving") {
		return newError(Network, DNS, "Volt couldn't find the host.", "Check the URL for typos, then try again.", err)
	}

	if strings.Contains(lower, "connection refused") {
		return newError(Network, ConnectionRefused, "Volt couldn't connect to the server.", "Check that the server is running, then try again.", err)
	}

	return newError(Network, Transport, "Volt couldn't complete the request.", "Check your connection and try again.", err)
}

func InvalidURL(value string) *Error {
	hint := "Enter a complete URL beginning with http:// or https://."
	trimmed := strings.TrimSpace(value)
	if trimmed != "" && !strings.Contains(trimmed, "://") {
		hint = fmt.Sprintf("Did you mean https://%s?", trimmed)
	}
	return newError(Validation, InvalidURLCode, "The URL is not valid.", hint, nil)
}

func ValidationError(code Code, message, hint string) *Error {
	return newError(Validation, code, message, hint, nil)
}

func FromStorage(err error) *Error {
	if typed, ok := asAppError(err); ok {
		return typed
	}

	lower := strings.ToLower(errorText(err))
	if strings.Contains(lower, "database is locked") || strings.Contains(lower, "database locked") || strings.Contains(lower, "database is busy") {
		return newError(Storage, DatabaseLocked, "Saved requests are busy right now.", "Wait a moment, then try again.", err)
	}
	if strings.Contains(lower, "i/o") || strings.Contains(lower, "no space") || strings.Contains(lower, "disk full") || strings.Contains(lower, "permission denied") {
		return newError(Storage, StorageIO, "Volt couldn't access saved requests.", "Check available disk space and permissions, then try again.", err)
	}
	return newError(Storage, StorageIO, "Volt couldn't update saved requests.", "Try again in a moment.", err)
}

func HTTPStatus(status int) *Error {
	switch {
	case status == 429:
		return newError(Network, HTTPClientError, "The server is rate limiting this request.", "Wait a moment before trying again or lower the load-test QPS.", nil)
	case status >= 500:
		return newError(Network, HTTPServerError, fmt.Sprintf("The server returned HTTP %d.", status), "Try again in a moment.", nil)
	default:
		return newError(Network, HTTPClientError, fmt.Sprintf("The server rejected the request (HTTP %d).", status), "Check the URL, headers, and body, then try again.", nil)
	}
}

// LoadTestFailure condenses a run's failure classes into one actionable status
// notification while preserving the complete per-class breakdown in the result pane.
func LoadTestFailure(failed int, classes map[string]int64) *Error {
	if classes[string(Timeout)] > 0 {
		return newError(
			Network,
			Timeout,
			fmt.Sprintf("Load test completed: %d requests failed (%d timed out).", failed, classes[string(Timeout)]),
			"Try again or increase the timeout.",
			nil,
		)
	}
	if classes[string(DNS)] > 0 {
		return newError(Network, DNS, fmt.Sprintf("Load test completed: %d requests failed because the host could not be found.", failed), "Check the URL for typos, then try again.", nil)
	}
	if classes[string(ConnectionRefused)] > 0 {
		return newError(Network, ConnectionRefused, fmt.Sprintf("Load test completed: %d requests could not connect to the server.", failed), "Check that the server is running, then try again.", nil)
	}
	if classes[string(HTTPServerError)] > 0 {
		return newError(Network, HTTPServerError, fmt.Sprintf("Load test completed: %d requests failed with server errors.", failed), "Try again in a moment.", nil)
	}
	return newError(Network, Transport, fmt.Sprintf("Load test completed: %d requests failed.", failed), "Check the URL and connection, then try again.", nil)
}

func ApplicationError() *Error {
	return newError(Application, Unexpected, "Volt recovered from an unexpected error.", "Press ? to see available keybindings, then try again.", nil)
}

func FromApplication(err error) *Error {
	if typed, ok := asAppError(err); ok {
		return typed
	}
	return newError(Application, Unexpected, "Volt couldn't complete that action.", "Press ? to see available keybindings, then try again.", err)
}

func OperationError(message, hint string) *Error {
	return newError(Application, Unexpected, message, hint, nil)
}

func ErrorClassLabel(class string) string {
	switch Code(class) {
	case Timeout:
		return "Timed out"
	case DNS:
		return "Host not found"
	case ConnectionRefused:
		return "Connection refused"
	case Transport:
		return "Connection errors"
	case HTTPClientError:
		return "HTTP 4xx responses"
	case HTTPServerError:
		return "HTTP 5xx responses"
	case "canceled":
		return "Canceled"
	default:
		return class
	}
}

func newError(category Category, code Code, message, hint string, cause error) *Error {
	return &Error{Category: category, Code: code, Message: message, Hint: hint, cause: cause}
}

func asAppError(err error) (*Error, bool) {
	var typed *Error
	if errors.As(err, &typed) {
		return typed, true
	}
	return nil, false
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
