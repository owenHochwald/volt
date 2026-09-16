package storage

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/owenHochwald/Volt/internal/http"
)

func TestSQLiteStorage_ServerCRUD(t *testing.T) {
	db := setupTestDB(t)

	server := &http.Server{Name: "local", BaseURL: "http://localhost:3000"}
	assert.NoError(t, db.SaveServer(server))
	assert.NotEqual(t, 0, server.ID)

	found, err := db.GetServerByName("local")
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:3000", found.BaseURL)

	missing, err := db.GetServerByName("nope")
	assert.NoError(t, err)
	assert.Zero(t, missing)

	assert.NoError(t, db.UpdateServerURL("local", "http://localhost:4000"))
	found, err = db.GetServerByName("local")
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:4000", found.BaseURL)

	assert.NoError(t, db.RenameServer("local", "dev"))
	found, err = db.GetServerByName("dev")
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:4000", found.BaseURL)

	servers, err := db.ListServers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(servers))

	assert.NoError(t, db.DeleteServer("dev"))
	servers, err = db.ListServers()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(servers))

	assert.Error(t, db.DeleteServer("dev"))
	assert.Error(t, db.UpdateServerURL("dev", "http://x"))
	assert.Error(t, db.RenameServer("dev", "prod"))
}

func TestSQLiteStorage_UpsertServer(t *testing.T) {
	db := setupTestDB(t)

	created, err := db.UpsertServer("prod", "https://api.example.com")
	assert.NoError(t, err)
	assert.NotEqual(t, 0, created.ID)

	updated, err := db.UpsertServer("prod", "https://api2.example.com")
	assert.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "https://api2.example.com", updated.BaseURL)

	servers, err := db.ListServers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(servers), "upsert on an existing name must not create a duplicate row")
}

func TestSQLiteStorage_GroupCRUD(t *testing.T) {
	db := setupTestDB(t)

	group := &http.Group{Name: "petstore"}
	assert.NoError(t, db.SaveGroup(group))
	assert.NotEqual(t, 0, group.ID)

	found, err := db.GetGroupByName("petstore")
	assert.NoError(t, err)
	assert.Zero(t, found.ServerID, "a freshly created group has no server")

	missing, err := db.GetGroupByName("nope")
	assert.NoError(t, err)
	assert.Zero(t, missing)

	// set-server requires the server to already exist.
	assert.Error(t, db.SetGroupServer("petstore", "prod"))

	_, err = db.UpsertServer("prod", "https://api.example.com")
	assert.NoError(t, err)
	assert.NoError(t, db.SetGroupServer("petstore", "prod"))

	found, err = db.GetGroupByName("petstore")
	assert.NoError(t, err)
	assert.NotZero(t, found.ServerID)

	assert.NoError(t, db.RenameGroup("petstore", "shop"))
	found, err = db.GetGroupByName("shop")
	assert.NoError(t, err)
	assert.NotZero(t, found.ServerID, "rename must preserve the server assignment")

	groups, err := db.ListGroups()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(groups))

	deleted, err := db.DeleteGroup("shop")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted, "an empty group deletes 0 requests")
	groups, err = db.ListGroups()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(groups))

	_, err = db.DeleteGroup("shop")
	assert.Error(t, err, "deleting an already-gone group must fail")
}

func TestSQLiteStorage_DeleteGroup_CascadesToItsRequests(t *testing.T) {
	db := setupTestDB(t)

	group, err := db.GetOrCreateGroup("petstore")
	assert.NoError(t, err)

	inGroup := &http.Request{Name: "in group", Method: http.GET, URL: "/pets", Headers: map[string]string{}, GroupID: &group.ID}
	assert.NoError(t, db.Save(inGroup))

	ungrouped := &http.Request{Name: "ungrouped", Method: http.GET, URL: "http://example.com", Headers: map[string]string{}}
	assert.NoError(t, db.Save(ungrouped))

	deleted, err := db.DeleteGroup("petstore")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	remaining, err := db.Load()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(remaining))
	assert.Equal(t, "ungrouped", remaining[0].Name)
}

func TestSQLiteStorage_GetOrCreateGroup(t *testing.T) {
	db := setupTestDB(t)

	first, err := db.GetOrCreateGroup("petstore")
	assert.NoError(t, err)

	second, err := db.GetOrCreateGroup("petstore")
	assert.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	groups, err := db.ListGroups()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(groups))
}

func TestSQLiteStorage_ResolveEffectiveURL(t *testing.T) {
	db := setupTestDB(t)

	// Absolute URLs pass through untouched, group or not.
	abs := &http.Request{Method: http.GET, URL: "http://example.com/x"}
	resolved, err := db.ResolveEffectiveURL(abs)
	assert.NoError(t, err)
	assert.Equal(t, "http://example.com/x", resolved)

	// A relative URL with no group is returned as-is (not resolvable).
	noGroup := &http.Request{Method: http.GET, URL: "/pets"}
	resolved, err = db.ResolveEffectiveURL(noGroup)
	assert.NoError(t, err)
	assert.Equal(t, "/pets", resolved)

	group, err := db.GetOrCreateGroup("petstore")
	assert.NoError(t, err)

	// Relative URL, group exists but has no server yet: still unresolved.
	grouped := &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID}
	resolved, err = db.ResolveEffectiveURL(grouped)
	assert.NoError(t, err)
	assert.Equal(t, "/pets", resolved)

	_, err = db.UpsertServer("petstore", "https://api.example.com/v1")
	assert.NoError(t, err)
	assert.NoError(t, db.SetGroupServer("petstore", "petstore"))

	resolved, err = db.ResolveEffectiveURL(grouped)
	assert.NoError(t, err)
	assert.Equal(t, "https://api.example.com/v1/pets", resolved)

	// Switching the group's server redirects every request in it.
	_, err = db.UpsertServer("petstore", "http://localhost:9000")
	assert.NoError(t, err)
	resolved, err = db.ResolveEffectiveURL(grouped)
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:9000/pets", resolved)
}

func TestSQLiteStorage_ServerHeaders(t *testing.T) {
	db := setupTestDB(t)

	server := &http.Server{Name: "prod", BaseURL: "https://api.example.com"}
	assert.NoError(t, db.SaveServer(server))

	found, err := db.GetServerByName("prod")
	assert.NoError(t, err)
	assert.Zero(t, len(found.Headers), "a freshly created server has no default headers")

	assert.NoError(t, db.UpdateServerHeaders("prod", map[string]string{"Authorization": "Bearer abc", "X-Env": "prod"}))

	found, err = db.GetServerByName("prod")
	assert.NoError(t, err)
	assert.Equal(t, "Bearer abc", found.Headers["Authorization"])
	assert.Equal(t, "prod", found.Headers["X-Env"])

	byID, err := db.GetServerByID(found.ID)
	assert.NoError(t, err)
	assert.Equal(t, found.Headers, byID.Headers)

	// Replacing headers overwrites the whole set, not a merge.
	assert.NoError(t, db.UpdateServerHeaders("prod", map[string]string{"X-Env": "staging"}))
	found, err = db.GetServerByName("prod")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(found.Headers))
	assert.Equal(t, "staging", found.Headers["X-Env"])

	assert.Error(t, db.UpdateServerHeaders("nope", map[string]string{"a": "b"}))
}

func TestSQLiteStorage_ResolveEffectiveServer(t *testing.T) {
	db := setupTestDB(t)

	// Neither ServerID nor GroupID set: no server.
	plain := &http.Request{Method: http.GET, URL: "/pets"}
	server, err := db.ResolveEffectiveServer(plain)
	assert.NoError(t, err)
	assert.Zero(t, server)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	assert.NoError(t, err)
	assert.NoError(t, db.UpdateServerHeaders("prod", map[string]string{"X-Env": "prod"}))

	// A direct ServerID resolves without needing a group.
	direct := &http.Request{Method: http.GET, URL: "/pets", ServerID: &prod.ID}
	server, err = db.ResolveEffectiveServer(direct)
	assert.NoError(t, err)
	assert.Equal(t, "https://api.example.com", server.BaseURL)
	assert.Equal(t, "prod", server.Headers["X-Env"])

	// A direct ServerID takes priority over the request's group's server.
	group, err := db.GetOrCreateGroup("petstore")
	assert.NoError(t, err)
	_, err = db.UpsertServer("staging", "https://staging.example.com")
	assert.NoError(t, err)
	assert.NoError(t, db.SetGroupServer("petstore", "staging"))

	both := &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID, ServerID: &prod.ID}
	server, err = db.ResolveEffectiveServer(both)
	assert.NoError(t, err)
	assert.Equal(t, "https://api.example.com", server.BaseURL, "a direct ServerID must win over the group's server")

	// With no ServerID, falls back to the group's server.
	viaGroup := &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID}
	server, err = db.ResolveEffectiveServer(viaGroup)
	assert.NoError(t, err)
	assert.Equal(t, "https://staging.example.com", server.BaseURL)
}

func TestSQLiteStorage_SaveLoadRoundTripsGroupID(t *testing.T) {
	db := setupTestDB(t)

	group, err := db.GetOrCreateGroup("petstore")
	assert.NoError(t, err)

	req := &http.Request{
		Name:    "list pets",
		Method:  http.GET,
		URL:     "/pets",
		Headers: map[string]string{},
		GroupID: &group.ID,
	}
	assert.NoError(t, db.Save(req))

	loaded, err := db.Load()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(loaded))
	assert.NotZero(t, loaded[0].GroupID)
	assert.Equal(t, group.ID, *loaded[0].GroupID)

	// A plain request (no group) round-trips with a nil GroupID.
	ungrouped := &http.Request{Name: "ungrouped", Method: http.GET, URL: "http://example.com", Headers: map[string]string{}}
	assert.NoError(t, db.Save(ungrouped))

	loaded, err = db.Load()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(loaded))
	for _, r := range loaded {
		if r.Name == "ungrouped" {
			assert.Zero(t, r.GroupID)
		}
	}
}

func TestSQLiteStorage_SaveLoadRoundTripsServerID(t *testing.T) {
	db := setupTestDB(t)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	assert.NoError(t, err)

	req := &http.Request{
		Name:     "list pets",
		Method:   http.GET,
		URL:      "/pets",
		Headers:  map[string]string{},
		ServerID: &prod.ID,
	}
	assert.NoError(t, db.Save(req))

	loaded, err := db.Load()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(loaded))
	assert.NotZero(t, loaded[0].ServerID)
	assert.Equal(t, prod.ID, *loaded[0].ServerID)

	// A plain request (no server) round-trips with a nil ServerID.
	unassigned := &http.Request{Name: "unassigned", Method: http.GET, URL: "http://example.com", Headers: map[string]string{}}
	assert.NoError(t, db.Save(unassigned))

	loaded, err = db.Load()
	assert.NoError(t, err)
	for _, r := range loaded {
		if r.Name == "unassigned" {
			assert.Zero(t, r.ServerID)
		}
	}
}
