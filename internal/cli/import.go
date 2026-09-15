package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/owenHochwald/Volt/internal/storage"
)

// ImportSummary reports the outcome of an OpenAPI import.
type ImportSummary struct {
	Imported int
	Skipped  int
	Warnings []string
}

// defaultServerName is the server -base-url is saved under for an
// ungrouped import, so it's still tracked in the servers table (and can
// later be assigned to a group) rather than being used once and forgotten.
const defaultServerName = "default"

// openDefaultDB opens Volt's saved-requests database at its standard
// location (~/.volt/volt.db), used by every subcommand that touches
// requests, groups, or servers.
func openDefaultDB() (*storage.SQLiteStorage, func(), error) {
	dbPath, err := storage.DefaultDBPath()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve database path: %w", err)
	}
	db, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, func() { db.Close() }, nil
}

// RunImportCLI parses `volt import` flags/args and imports the given spec
// into Volt's saved-requests database.
func RunImportCLI(args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	baseURL := fs.String("base-url", "", "Set/override the target group's server URL (or, ungrouped, save it as the \"default\" server)")
	group := fs.String("group", "", "Import into this request group (created if missing)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: volt import [-group <name>] [-base-url <url>] <openapi-file>")
	}
	specPath := fs.Arg(0)

	db, closeDB, err := openDefaultDB()
	if err != nil {
		return err
	}
	defer closeDB()

	summary, err := RunImport(specPath, *baseURL, *group, db)
	if err != nil {
		return err
	}

	fmt.Printf("Imported %d request(s), skipped %d already saved.\n", summary.Imported, summary.Skipped)
	for _, w := range summary.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	return nil
}

// RunImport reads an OpenAPI spec from specPath and saves one Request per
// supported operation into db, skipping any group+method+URL combination
// already stored.
//
// When groupName is non-empty, requests are tagged with that group
// (created if it doesn't exist yet) and stored as bare paths rather than
// absolute URLs, so they resolve dynamically against the group's server.
// baseURLOverride then sets/updates that group's server instead of
// prefixing the URLs directly; if the group has no server yet and the spec
// itself declares an absolute one, that's adopted automatically.
//
// With no groupName, request URLs are still absolute (baseURLOverride, or
// the spec's own server, prefixed onto every path) - but if
// baseURLOverride was explicitly passed, it's also saved to the servers
// table (as defaultServerName) rather than used once and discarded, so it
// shows up in `volt server list` and can be assigned to a group later.
func RunImport(specPath, baseURLOverride, groupName string, db *storage.SQLiteStorage) (*ImportSummary, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", specPath, err)
	}
	isYAML := looksLikeYAML(specPath, data)

	var groupID *int64
	if groupName != "" {
		group, err := db.GetOrCreateGroup(groupName)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare group %q: %w", groupName, err)
		}
		groupID = &group.ID

		if baseURLOverride != "" {
			server, err := db.UpsertServer(groupName, baseURLOverride)
			if err != nil {
				return nil, fmt.Errorf("failed to configure server for group %q: %w", groupName, err)
			}
			if err := db.SetGroupServer(groupName, server.Name); err != nil {
				return nil, fmt.Errorf("failed to assign server to group %q: %w", groupName, err)
			}
		} else if group.ServerID == nil {
			// New/unconfigured group: adopt the spec's own server, if any,
			// so the group is usable without a separate `volt server` step.
			if specServerURL, err := SpecServerURL(data, isYAML); err == nil && specServerURL != "" {
				if server, err := db.UpsertServer(groupName, specServerURL); err == nil {
					_ = db.SetGroupServer(groupName, server.Name)
				}
			}
		}
	} else if baseURLOverride != "" {
		if _, err := db.UpsertServer(defaultServerName, baseURLOverride); err != nil {
			return nil, fmt.Errorf("failed to save server %q: %w", defaultServerName, err)
		}
	}

	requests, warnings, err := ParseOpenAPIURLs(data, isYAML, baseURLOverride, groupName != "")
	if err != nil {
		return nil, err
	}

	existing, err := db.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to read existing requests: %w", err)
	}
	seen := make(map[string]bool, len(existing))
	for _, r := range existing {
		seen[dedupKey(r.GroupID, r.Method, r.URL)] = true
	}

	summary := &ImportSummary{Warnings: warnings}
	for _, req := range requests {
		req.GroupID = groupID
		key := dedupKey(req.GroupID, req.Method, req.URL)
		if seen[key] {
			summary.Skipped++
			continue
		}
		if err := db.Save(req); err != nil {
			summary.Warnings = append(summary.Warnings, fmt.Sprintf("failed to save %s: %v", key, err))
			continue
		}
		seen[key] = true
		summary.Imported++
	}

	return summary, nil
}

// dedupKey scopes method+URL dedup by group, since the same relative path
// (e.g. "/pets") legitimately means different things in different groups.
func dedupKey(groupID *int64, method, url string) string {
	gid := int64(0)
	if groupID != nil {
		gid = *groupID
	}
	return fmt.Sprintf("%d|%s|%s", gid, method, url)
}

func looksLikeYAML(path string, data []byte) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	case ".json":
		return false
	}
	trimmed := strings.TrimSpace(string(data))
	return !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[")
}
