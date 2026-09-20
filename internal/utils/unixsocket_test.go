package utils

import "testing"

func TestParseUnixSocketURL(t *testing.T) {
	tests := []struct {
		name           string
		rawURL         string
		wantSocketPath string
		wantReqPath    string
		wantOK         bool
	}{
		{
			name:           "socket path and request path",
			rawURL:         "unix:///var/run/docker.sock:/containers/json",
			wantSocketPath: "/var/run/docker.sock",
			wantReqPath:    "/containers/json",
			wantOK:         true,
		},
		{
			name:           "bare socket path defaults to root",
			rawURL:         "unix:///var/run/docker.sock",
			wantSocketPath: "/var/run/docker.sock",
			wantReqPath:    "/",
			wantOK:         true,
		},
		{
			name:           "request path with query string",
			rawURL:         "unix:///tmp/app.sock:/status?verbose=true",
			wantSocketPath: "/tmp/app.sock",
			wantReqPath:    "/status?verbose=true",
			wantOK:         true,
		},
		{
			name:   "not a unix URL",
			rawURL: "https://api.example.com/widgets",
			wantOK: false,
		},
		{
			name:   "relative path",
			rawURL: "/widgets",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			socketPath, reqPath, ok := ParseUnixSocketURL(tt.rawURL)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if socketPath != tt.wantSocketPath {
				t.Errorf("socketPath = %q, want %q", socketPath, tt.wantSocketPath)
			}
			if reqPath != tt.wantReqPath {
				t.Errorf("requestPath = %q, want %q", reqPath, tt.wantReqPath)
			}
		})
	}
}

func TestIsUnixSocketURL(t *testing.T) {
	if !IsUnixSocketURL("unix:///tmp/app.sock:/health") {
		t.Error("expected a unix:// URL to be recognized")
	}
	if IsUnixSocketURL("https://api.example.com") {
		t.Error("expected an https:// URL not to be recognized as a unix socket URL")
	}
}
