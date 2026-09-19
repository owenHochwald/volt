package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/storage"
)

// withTestDB points openDefaultDB (via DBPathOverride) at a fresh
// temporary database for the duration of t, restoring the previous
// override afterward - shared by every RunServerCLI test below so none
// of them ever touch a real ~/.volt/volt.db.
func withTestDB(t *testing.T) {
	t.Helper()
	previous := DBPathOverride
	DBPathOverride = filepath.Join(t.TempDir(), "volt.db")
	t.Cleanup(func() { DBPathOverride = previous })
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return string(data)
}

// TestRunServerCLI_SetVarVarsUnsetVar is a regression test for Phase 1 of
// "environment variables": a server's {{name}} placeholder values (see
// http.SubstituteVariables) are manageable from the CLI, mirroring the
// existing set-header/headers/unset-header commands exactly.
func TestRunServerCLI_SetVarVarsUnsetVar(t *testing.T) {
	withTestDB(t)

	if err := RunServerCLI([]string{"add", "prod", "https://api.example.com"}); err != nil {
		t.Fatalf("failed to add server: %v", err)
	}

	if err := RunServerCLI([]string{"set-var", "prod", "token", "abc123"}); err != nil {
		t.Fatalf("failed to set variable: %v", err)
	}
	if err := RunServerCLI([]string{"set-var", "prod", "host", "api.example.com"}); err != nil {
		t.Fatalf("failed to set variable: %v", err)
	}

	out := captureStdout(t, func() {
		if err := RunServerCLI([]string{"vars", "prod"}); err != nil {
			t.Fatalf("failed to list variables: %v", err)
		}
	})
	if !strings.Contains(out, "host = api.example.com") || !strings.Contains(out, "token = abc123") {
		t.Errorf("expected both variables listed, got:\n%s", out)
	}

	db, err := storage.NewSQLiteStorage(DBPathOverride)
	if err != nil {
		t.Fatalf("failed to reopen db: %v", err)
	}
	defer db.Close()
	server, err := db.GetServerByName("prod")
	if err != nil {
		t.Fatalf("failed to load server: %v", err)
	}
	// "host" sorts before "token" (see sortedKeys), so index 1 is "host".
	if err := RunServerCLI([]string{"unset-var", "prod", "1"}); err != nil {
		t.Fatalf("failed to unset variable: %v", err)
	}

	server, err = db.GetServerByName("prod")
	if err != nil {
		t.Fatalf("failed to reload server: %v", err)
	}
	if _, ok := server.Variables["host"]; ok {
		t.Error("expected \"host\" to be removed")
	}
	if server.Variables["token"] != "abc123" {
		t.Errorf("expected \"token\" to survive, got %+v", server.Variables)
	}
}

func TestRunServerCLI_Vars_NoVariables(t *testing.T) {
	withTestDB(t)
	if err := RunServerCLI([]string{"add", "prod", "https://api.example.com"}); err != nil {
		t.Fatalf("failed to add server: %v", err)
	}

	out := captureStdout(t, func() {
		if err := RunServerCLI([]string{"vars", "prod"}); err != nil {
			t.Fatalf("failed to list variables: %v", err)
		}
	})
	if !strings.Contains(out, "has no variables") {
		t.Errorf("expected a \"no variables\" message, got:\n%s", out)
	}
}

func TestRunServerCLI_UnsetVar_InvalidIndex(t *testing.T) {
	withTestDB(t)
	if err := RunServerCLI([]string{"add", "prod", "https://api.example.com"}); err != nil {
		t.Fatalf("failed to add server: %v", err)
	}
	if err := RunServerCLI([]string{"set-var", "prod", "token", "abc123"}); err != nil {
		t.Fatalf("failed to set variable: %v", err)
	}

	if err := RunServerCLI([]string{"unset-var", "prod", "5"}); err == nil {
		t.Error("expected an error for an out-of-range index")
	}
}
