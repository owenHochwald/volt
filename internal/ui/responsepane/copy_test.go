package responsepane

import (
	"errors"
	"testing"
)

func TestCopyCommandPropagatesClipboardError(t *testing.T) {
	expected := errors.New("clipboard unavailable")
	cmd := copyCommand("response", func(string) error {
		return expected
	})

	msg := cmd()
	result, ok := msg.(ResponseCopiedMsg)
	if !ok {
		t.Fatalf("message type = %T, want ResponseCopiedMsg", msg)
	}
	if !errors.Is(result.Err, expected) {
		t.Fatalf("error = %v, want %v", result.Err, expected)
	}
}
