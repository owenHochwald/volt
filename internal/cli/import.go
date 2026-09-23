package cli

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/owenHochwald/Volt/internal/storage"
)

// ImportSummary reports the outcome of an OpenAPI import.
type ImportSummary struct {
	Imported int
	// Updated counts requests that already existed (same group+method+url -
	// see dedupKey) and were overwritten in place because force was set;
	// always 0 without force, where a match is Skipped instead.
	Updated  int
	Skipped  int
	Warnings []string
}

// defaultServerName is the server -base-url is saved under for an
// ungrouped import, so it's still tracked in the servers table (and can
// later be assigned to a group) rather than being used once and forgotten.
const defaultServerName = "default"

// DBPathOverride, when non-empty, is used in place of Volt's standard
// database location (~/.volt/volt.db) by every subcommand that opens one
// (see openDefaultDB) - set from main via the top-level --db flag, so a
// user (or a test) can point the whole CLI at a different file without
// touching their real data.
var DBPathOverride string

// openDefaultDB opens Volt's saved-requests database - DBPathOverride if
// set, otherwise its standard location (~/.volt/volt.db) - used by every
// subcommand that touches requests, groups, or servers.
func openDefaultDB() (*storage.SQLiteStorage, func(), error) {
	dbPath := DBPathOverride
	if dbPath == "" {
		var err error
		dbPath, err = storage.DefaultDBPath()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve database path: %w", err)
		}
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
	force := fs.Bool("force", false, "Overwrite already-saved requests (same group+method+url) with the spec's current name/headers/body instead of skipping them")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: volt import [-group <name>] [-base-url <url>] [-force] <openapi-file-or-url>")
	}
	specPath := fs.Arg(0)

	db, closeDB, err := openDefaultDB()
	if err != nil {
		return err
	}
	defer closeDB()

	summary, err := RunImport(specPath, *baseURL, *group, *force, db)
	if err != nil {
		return err
	}

	fmt.Printf("Imported %d request(s), updated %d, skipped %d unchanged.\n", summary.Imported, summary.Updated, summary.Skipped)
	for _, w := range summary.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	return nil
}

// RunImport reads an OpenAPI spec from specPath and saves one Request per
// supported operation into db, skipping any group+method+URL combination
// already stored - or, with force set, overwriting that existing request's
// name/headers/body with what the spec currently says instead of skipping
// it (see UpdateRequestSchema). A forced update never touches the
// request's group/server association, and never creates a duplicate row -
// it's the same group+method+url match (see dedupKey) that decides Skipped
// vs Imported without force, just resolved differently.
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
func RunImport(specPath, baseURLOverride, groupName string, force bool, db *storage.SQLiteStorage) (*ImportSummary, error) {
	data, isYAML, err := readSpec(specPath)
	if err != nil {
		return nil, err
	}

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
	// existingIDs maps a request's identity (see dedupKey) to its row ID,
	// so a force update knows exactly which row to overwrite - a plain
	// seen-or-not set (as before force existed) isn't enough for that.
	existingIDs := make(map[string]int64, len(existing))
	for _, r := range existing {
		existingIDs[dedupKey(r.GroupID, r.Method, r.URL)] = r.ID
	}

	summary := &ImportSummary{Warnings: warnings}
	for _, req := range requests {
		req.GroupID = groupID
		key := dedupKey(req.GroupID, req.Method, req.URL)
		if id, ok := existingIDs[key]; ok {
			if !force {
				summary.Skipped++
				continue
			}
			if err := db.UpdateRequestSchema(id, req.Name, req.Headers, req.Body, req.PathParamSchema, req.QueryParamSchema, req.Description); err != nil {
				summary.Warnings = append(summary.Warnings, fmt.Sprintf("failed to update %s: %v", key, err))
				continue
			}
			summary.Updated++
			continue
		}
		if err := db.Save(req); err != nil {
			summary.Warnings = append(summary.Warnings, fmt.Sprintf("failed to save %s: %v", key, err))
			continue
		}
		existingIDs[key] = req.ID
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

// specFetchTimeout bounds how long readSpec waits for a spec to download
// over the network - generous enough for a large spec on a slow
// connection, but finite so an unreachable or stalled server doesn't hang
// `volt import` (or the TUI import modal's own Cmd) forever.
const specFetchTimeout = 30 * time.Second

// maxSpecSize caps how much of a fetched response readSpec will read - an
// OpenAPI spec is normally at most a few hundred KB of YAML/JSON; this
// just stops a misbehaving or malicious server from streaming an
// unbounded body into memory.
const maxSpecSize = 25 * 1024 * 1024 // 25 MiB

// readSpec loads specPath's raw bytes and reports whether it looks like
// YAML (as opposed to JSON) - from a local file, or, when specPath is an
// http(s):// URL, fetched over the network instead (a spec published at a
// service's own /openapi.yaml endpoint doesn't have to be downloaded by
// hand first).
func readSpec(specPath string) (data []byte, isYAML bool, err error) {
	if strings.HasPrefix(specPath, "http://") || strings.HasPrefix(specPath, "https://") {
		return fetchSpec(specPath)
	}
	data, err = os.ReadFile(specPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read %s: %w", specPath, err)
	}
	return data, looksLikeYAML(specPath, data), nil
}

// fetchSpec downloads specURL's body over HTTP(S) - the readSpec branch
// for a specPath that's itself a URL rather than a local file path.
func fetchSpec(specURL string) (data []byte, isYAML bool, err error) {
	client := &http.Client{Timeout: specFetchTimeout}
	resp, err := client.Get(specURL)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch %s: %w", specURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("failed to fetch %s: server returned %s", specURL, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSpecSize))
	if err != nil {
		return nil, false, fmt.Errorf("failed to read response body from %s: %w", specURL, err)
	}
	return body, looksLikeYAML(specURL, body), nil
}

// looksLikeYAML guesses a spec's format from path's extension - stripped
// of any query string/fragment first, so a URL like
// ".../openapi.yaml?ref=main" is still recognized as ".yaml" rather than
// falling through to the content-based fallback below - falling back to
// sniffing data itself (valid JSON always starts with "{" or "["; an
// OpenAPI YAML document normally doesn't) when the extension is missing or
// unrecognized, e.g. a URL with no file extension at all.
func looksLikeYAML(path string, data []byte) bool {
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	case ".json":
		return false
	}
	trimmed := strings.TrimSpace(string(data))
	return !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[")
}
