package requestpane

import (
	stdhttp "net/http"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/authjar"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func setupTestPane(t *testing.T) (RequestPane, *storage.SQLiteStorage) {
	t.Helper()
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pane := SetupRequestPane(db, keybindings.DefaultKeyMap())
	return pane, db
}

func TestRequestPane_ResolvedRequest_NoGroup(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.Request = &http.Request{Method: http.GET, URL: "http://example.com/x"}

	resolved := pane.resolvedRequest()
	if resolved != pane.Request {
		t.Errorf("expected the same request when there's no group, got a different pointer/value: %+v", resolved)
	}
}

func TestRequestPane_ResolvedRequest_GroupWithServer(t *testing.T) {
	pane, db := setupTestPane(t)

	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("petstore", "https://api.example.com/v1"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.SetGroupServer("petstore", "petstore"); err != nil {
		t.Fatalf("failed to assign server: %v", err)
	}

	pane.Request = &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/v1/pets" {
		t.Errorf("expected resolved URL to use the group's server, got %q", resolved.URL)
	}
	// The stored request must stay untouched (relative), so editing/saving
	// keeps following the group's server rather than freezing an absolute URL.
	if pane.Request.URL != "/pets" {
		t.Errorf("expected the original request to remain relative, got %q", pane.Request.URL)
	}
}

func TestRequestPane_ResolvedRequest_GroupWithoutServer(t *testing.T) {
	pane, db := setupTestPane(t)

	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	pane.Request = &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID}

	resolved := pane.resolvedRequest()
	if resolved.URL != "/pets" {
		t.Errorf("expected the URL to stay relative when the group has no server, got %q", resolved.URL)
	}
}

// TestRequestPane_ResolvedRequest_DirectServer covers a request that picks a
// server directly (via ServerID) rather than through a group - its relative
// URL must resolve against that server, and the server's default headers
// must be merged in underneath the request's own.
func TestRequestPane_ResolvedRequest_DirectServer(t *testing.T) {
	pane, db := setupTestPane(t)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.UpdateServerHeaders("prod", map[string]string{"X-Env": "prod", "Authorization": "Bearer server-token"}); err != nil {
		t.Fatalf("failed to set server headers: %v", err)
	}

	pane.Request = &http.Request{
		Method:   http.GET,
		URL:      "/pets",
		Headers:  map[string]string{"Authorization": "Bearer request-token"},
		ServerID: &prod.ID,
	}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/pets" {
		t.Errorf("expected the URL to resolve against the directly-selected server, got %q", resolved.URL)
	}
	if resolved.Headers["X-Env"] != "prod" {
		t.Errorf("expected the server's default header to be merged in, got %+v", resolved.Headers)
	}
	if resolved.Headers["Authorization"] != "Bearer request-token" {
		t.Errorf("expected the request's own header to win over the server's default, got %+v", resolved.Headers)
	}
	if pane.Request.URL != "/pets" {
		t.Errorf("expected the stored request to stay relative, got %q", pane.Request.URL)
	}
}

// TestRequestPane_ResolvedRequest_ServerIDOverridesGroup covers a request
// that has both a group and a direct ServerID - the direct selection must
// win, letting a single request point somewhere different from the rest of
// its group without leaving the group.
func TestRequestPane_ResolvedRequest_ServerIDOverridesGroup(t *testing.T) {
	pane, db := setupTestPane(t)

	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("staging", "https://staging.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.SetGroupServer("petstore", "staging"); err != nil {
		t.Fatalf("failed to assign group server: %v", err)
	}
	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	pane.Request = &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID, ServerID: &prod.ID}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/pets" {
		t.Errorf("expected the direct ServerID to win over the group's server, got %q", resolved.URL)
	}
}

// TestRequestPane_ServerSelector_SyncsRequestServerID covers the
// ServerSelector field itself: cycling to a server name and syncing must
// set the request's ServerID, and cycling back to "(none)" must clear it.
// Uses a grouped request - the field is only shown/populated for one (see
// ShowServerSelector); an ungrouped ("clean") request has no server field
// at all, see TestRebuildServerSelector_UngroupedRequestHasNoServerField.
func TestRequestPane_ServerSelector_SyncsRequestServerID(t *testing.T) {
	pane, db := setupTestPane(t)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if err := db.AddGroupServer("petstore", "prod"); err != nil {
		t.Fatalf("failed to add group server: %v", err)
	}

	req := &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID}
	pane.reinitRequestPane(req)

	if pane.ServerSelector.Value() != noServerOption {
		t.Fatalf("expected a request with no ServerID to start on %q, got %q", noServerOption, pane.ServerSelector.Value())
	}

	pane.ServerSelector.Next()
	if pane.ServerSelector.Value() != "prod" {
		t.Fatalf("expected the only configured server to be next in the cycle, got %q", pane.ServerSelector.Value())
	}

	pane.syncRequest()
	if pane.Request.ServerID == nil || *pane.Request.ServerID != prod.ID {
		t.Errorf("expected syncRequest to set ServerID from the selector, got %+v", pane.Request.ServerID)
	}

	pane.ServerSelector.Next()
	if pane.ServerSelector.Value() != noServerOption {
		t.Fatalf("expected cycling past the only server to return to %q, got %q", noServerOption, pane.ServerSelector.Value())
	}
	pane.syncRequest()
	if pane.Request.ServerID != nil {
		t.Errorf("expected syncRequest to clear ServerID for %q, got %+v", noServerOption, pane.Request.ServerID)
	}
}

// TestRequestPane_ReinitRequestPane_ServerSelectorReflectsServerID covers
// loading a (grouped) request that already has a ServerID: the selector
// must start on that server's name, not "(none)".
func TestRequestPane_ReinitRequestPane_ServerSelectorReflectsServerID(t *testing.T) {
	pane, db := setupTestPane(t)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if err := db.AddGroupServer("petstore", "prod"); err != nil {
		t.Fatalf("failed to add group server: %v", err)
	}

	req := &http.Request{Method: http.GET, URL: "/pets", ServerID: &prod.ID, GroupID: &group.ID}
	pane.reinitRequestPane(req)

	if pane.ServerSelector.Value() != "prod" {
		t.Errorf("expected the selector to start on the request's server, got %q", pane.ServerSelector.Value())
	}
}

// TestRequestPane_ServerSelector_CyclesViaFocusManager reproduces the actual
// key-handling path (FocusManager.Current() + HandleInput), not a direct
// call on the ServerSelector field - reinitRequestPane always rebuilds a
// fresh ServerSelector, but for a request with no path/body params (so
// syncPathParams/syncBodyParams don't themselves trigger a focus-cycle
// rebuild), FocusManager could be left pointing at the *previous*,
// now-orphaned ServerSelector instance. Cycling it via the real input path
// must still reach the field View() actually renders.
func TestRequestPane_ServerSelector_CyclesViaFocusManager(t *testing.T) {
	pane, db := setupTestPane(t)

	if _, err := db.UpsertServer("prod", "https://api.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if err := db.AddGroupServer("petstore", "prod"); err != nil {
		t.Fatalf("failed to add group server: %v", err)
	}

	// A (grouped, so it gets a server field - see ShowServerSelector)
	// request with no path params and no body schema - both
	// syncPathParams and syncBodyParams are no-ops for it.
	req := &http.Request{Method: http.GET, URL: "/pets", Headers: map[string]string{}, GroupID: &group.ID}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	// reinitRequestPane defaults focus to method+URL (see
	// defaultFocusIndex), not the server field (index 0) - move there
	// explicitly; what this test actually exercises is cycling the
	// currently-focused field via the real input path, not which field
	// ends up focused by default.
	pane.FocusManager.FocusIndex(0)
	if pane.FocusManager.CurrentIndex() != 0 {
		t.Fatalf("expected the server selector (first field) focused, got index %d", pane.FocusManager.CurrentIndex())
	}

	result, _ := pane.currentMode.HandleInput(&pane, tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}))

	if result.ServerSelector.Value() != "prod" {
		t.Errorf("expected cycling via the real input path to change the selector actually rendered, got %q", result.ServerSelector.Value())
	}
}

// TestRequestPane_SyncRequest_ArrayOnlyBodyDoesNotReparseHiddenTextarea is a
// regression test for a real bug found against an actual OpenAPI spec: a
// body schema with *only* an array-of-objects field (no plain scalar
// properties) left BodyParams empty while ArrayParams held the array -
// syncRequest checked "len(BodyParams) == 0" alone to decide whether the
// freeform Body textarea was in play, so it wrongly reparsed the *hidden*
// Body textarea's raw schema-template JSON as bogus "key=value" pairs on
// every send attempt (surfacing as several nonsense "Invalid key-value
// pair" errors that never went away no matter what was typed into the
// array's actual fields, since nothing was editing that hidden textarea).
func TestRequestPane_SyncRequest_ArrayOnlyBodyDoesNotReparseHiddenTextarea(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"tags": {Type: "array", ItemsRefName: "Tag", Items: tagItemSchema()},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 0 {
		t.Fatalf("expected no scalar body params, got %+v", pane.BodyParams)
	}
	if len(pane.ArrayParams) != 1 {
		t.Fatalf("expected 1 array param, got %+v", pane.ArrayParams)
	}

	pane.syncRequest()

	if len(pane.ParseErrors) != 0 {
		t.Errorf("expected no spurious parse errors from the hidden Body textarea, got %v", pane.ParseErrors)
	}
	if pane.Request.Body != req.Body {
		t.Errorf("expected the stored body template to survive a sync tick untouched, got %q", pane.Request.Body)
	}
}

// TestRequestPane_ResolvedRequest_AddsContentTypeForSchemaBody is a
// regression test for a real feature request: a request built from an
// imported JSON body schema (BodyParams/ArrayParams populated) went out
// with no Content-Type header at all, leaving the server to guess how to
// parse a body that's always actually JSON (see BuildRequestBody).
func TestRequestPane_ResolvedRequest_AddsContentTypeForSchemaBody(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {Type: "string"}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.BodyParams[0].Input.SetValue("widget-1")

	resolved := pane.resolvedRequest()

	if got := resolved.Headers["Content-Type"]; got != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q (headers=%v)", got, resolved.Headers)
	}
	if _, ok := pane.Request.Headers["Content-Type"]; ok {
		t.Error("expected the stored request template to stay untouched, not gain Content-Type")
	}
}

// TestRequestPane_ResolvedRequest_NeverOverridesExplicitContentType
// verifies a Content-Type the user already set (any casing) is left
// alone - auto-adding one is only a default for when nothing was said,
// never an override of a deliberate choice.
func TestRequestPane_ResolvedRequest_NeverOverridesExplicitContentType(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {Type: "string"}})
	req := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/widgets",
		Body:    body,
		Headers: map[string]string{"content-type": "application/vnd.custom+json"},
	}
	pane.reinitRequestPane(req)
	pane.BodyParams[0].Input.SetValue("widget-1")

	resolved := pane.resolvedRequest()

	if len(resolved.Headers) != 1 {
		t.Fatalf("expected exactly 1 header (no duplicate differently-cased Content-Type), got %v", resolved.Headers)
	}
	if got := resolved.Headers["content-type"]; got != "application/vnd.custom+json" {
		t.Errorf("expected the user's own content-type to survive untouched, got %q", got)
	}
}

// TestRequestPane_ResolvedRequest_NoContentTypeForFreeformBody verifies
// the auto-added Content-Type is scoped to a schema-driven body
// (BodyParams/ArrayParams) - a request with no known schema (the freeform
// Body textarea) is left exactly as before, since Volt can't actually
// tell what content type free-typed text is meant to be.
func TestRequestPane_ResolvedRequest_NoContentTypeForFreeformBody(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.Request = &http.Request{Method: http.POST, URL: "http://example.com/x", Headers: map[string]string{}}

	resolved := pane.resolvedRequest()

	if _, ok := resolved.Headers["Content-Type"]; ok {
		t.Errorf("expected no auto-added Content-Type for a freeform-body request, got %v", resolved.Headers)
	}
}

// TestRequestPane_ResolvedRequest_InjectsJarHeaders_NoServer verifies the
// jar's remembered headers (see internal/authjar) get injected for an
// ad-hoc, absolute-URL request with no configured server at all - keyed
// by host alone, not ServerID/GroupID.
func TestRequestPane_ResolvedRequest_InjectsJarHeaders_NoServer(t *testing.T) {
	pane, _ := setupTestPane(t)
	jar := authjar.NewStore()
	jar.RememberAuthHeader("api.example.com", "Authorization", "Bearer jar-token")
	pane.Jar = jar

	pane.Request = &http.Request{Method: http.GET, URL: "https://api.example.com/pets"}

	resolved := pane.resolvedRequest()
	if resolved.Headers["Authorization"] != "Bearer jar-token" {
		t.Errorf("expected the jar's Authorization header to be injected, got %+v", resolved.Headers)
	}
}

// TestRequestPane_ResolvedRequest_RequestHeaderOverridesJar verifies the
// jar is the lowest-precedence layer - a header the request already sets
// itself must win, the same way it already wins over a server's default
// header.
func TestRequestPane_ResolvedRequest_RequestHeaderOverridesJar(t *testing.T) {
	pane, _ := setupTestPane(t)
	jar := authjar.NewStore()
	jar.RememberAuthHeader("api.example.com", "Authorization", "Bearer jar-token")
	pane.Jar = jar

	pane.Request = &http.Request{
		Method:  http.GET,
		URL:     "https://api.example.com/pets",
		Headers: map[string]string{"Authorization": "Bearer explicit-token"},
	}

	resolved := pane.resolvedRequest()
	if resolved.Headers["Authorization"] != "Bearer explicit-token" {
		t.Errorf("expected the request's own header to win over the jar, got %+v", resolved.Headers)
	}
}

// TestRequestPane_ResolvedRequest_ServerHeaderOverridesJar mirrors
// TestRequestPane_ResolvedRequest_RequestHeaderOverridesJar for a
// server's own configured header.
func TestRequestPane_ResolvedRequest_ServerHeaderOverridesJar(t *testing.T) {
	pane, db := setupTestPane(t)
	jar := authjar.NewStore()
	jar.RememberAuthHeader("api.example.com", "Authorization", "Bearer jar-token")
	pane.Jar = jar

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.UpdateServerHeaders("prod", map[string]string{"Authorization": "Bearer server-token"}); err != nil {
		t.Fatalf("failed to set server headers: %v", err)
	}

	pane.Request = &http.Request{Method: http.GET, URL: "/pets", ServerID: &prod.ID}

	resolved := pane.resolvedRequest()
	if resolved.Headers["Authorization"] != "Bearer server-token" {
		t.Errorf("expected the server's header to win over the jar, got %+v", resolved.Headers)
	}
}

// TestRequestPane_ResolvedRequest_JarCookiesInjectedAlongsideServerHeaders
// verifies the jar and server headers both apply together when they
// don't collide - a cookie remembered from an earlier login response
// riding alongside the server's own unrelated default header.
func TestRequestPane_ResolvedRequest_JarCookiesInjectedAlongsideServerHeaders(t *testing.T) {
	pane, db := setupTestPane(t)
	jar := authjar.NewStore()
	jar.RememberCookies("api.example.com", []*stdhttp.Cookie{{Name: "session", Value: "abc123"}})
	pane.Jar = jar

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.UpdateServerHeaders("prod", map[string]string{"X-Env": "prod"}); err != nil {
		t.Fatalf("failed to set server headers: %v", err)
	}

	pane.Request = &http.Request{Method: http.GET, URL: "/pets", ServerID: &prod.ID}

	resolved := pane.resolvedRequest()
	if resolved.Headers["X-Env"] != "prod" {
		t.Errorf("expected the server header to still apply, got %+v", resolved.Headers)
	}
	if resolved.Headers["Cookie"] != "session=abc123" {
		t.Errorf("expected the jar's cookie to be injected, got %+v", resolved.Headers)
	}
}

// TestRequestPane_ResolvedRequest_NoJarHeaders_UnaffectedRequestStaysSame
// verifies an empty jar is still a true no-op, matching
// TestRequestPane_ResolvedRequest_NoGroup's same-pointer assertion.
func TestRequestPane_ResolvedRequest_NoJarHeaders_UnaffectedRequestStaysSame(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.Jar = authjar.NewStore()
	pane.Request = &http.Request{Method: http.GET, URL: "http://example.com/x"}

	resolved := pane.resolvedRequest()
	if resolved != pane.Request {
		t.Errorf("expected the same request when the jar has nothing for this host, got a different pointer/value: %+v", resolved)
	}
}

func TestMergeHeaders_OverrideWinsCaseInsensitively(t *testing.T) {
	base := map[string]string{"Cookie": "session=old", "X-Env": "prod"}
	override := map[string]string{"cookie": "session=new"}

	merged := mergeHeaders(base, override)

	if len(merged) != 2 {
		t.Fatalf("expected 2 headers (no duplicate differently-cased Cookie), got %v", merged)
	}
	if merged["cookie"] != "session=new" {
		t.Errorf("expected override's cookie value to win, got %+v", merged)
	}
	if merged["X-Env"] != "prod" {
		t.Errorf("expected base's unrelated header to survive, got %+v", merged)
	}
}

// TestRequestPane_ResolvedRequest_SubstitutesServerVariables is a
// regression test for Phase 1 of "environment variables": {{name}}
// placeholders in the URL, headers, and body all resolve from the
// effective server's own Variables (see http.SubstituteVariables) - a
// server plays the role of an "environment" here, so dev/staging/prod
// each get their own variable values just by being different servers.
func TestRequestPane_ResolvedRequest_SubstitutesServerVariables(t *testing.T) {
	pane, db := setupTestPane(t)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.UpdateServerVariables("prod", map[string]string{"token": "prod-secret", "widget_id": "42"}); err != nil {
		t.Fatalf("failed to set server variables: %v", err)
	}

	pane.Request = &http.Request{
		Method:   http.POST,
		URL:      "/widgets/{{widget_id}}",
		Body:     `{"note": "using {{token}}"}`,
		Headers:  map[string]string{"Authorization": "Bearer {{token}}"},
		ServerID: &prod.ID,
	}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/widgets/42" {
		t.Errorf("expected the URL placeholder to resolve, got %q", resolved.URL)
	}
	if resolved.Headers["Authorization"] != "Bearer prod-secret" {
		t.Errorf("expected the header placeholder to resolve, got %+v", resolved.Headers)
	}
	if resolved.Body != `{"note": "using prod-secret"}` {
		t.Errorf("expected the body placeholder to resolve, got %q", resolved.Body)
	}
	// The stored request keeps its raw placeholders, same as PathParams.
	if pane.Request.URL != "/widgets/{{widget_id}}" {
		t.Errorf("expected the stored request's URL to keep its placeholder, got %q", pane.Request.URL)
	}
}

// TestRequestPane_ResolvedRequest_UnknownVariableStaysLiteral verifies a
// typo'd or not-yet-defined variable name is visibly wrong in the
// resolved request rather than silently disappearing.
func TestRequestPane_ResolvedRequest_UnknownVariableStaysLiteral(t *testing.T) {
	pane, db := setupTestPane(t)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.UpdateServerVariables("prod", map[string]string{"token": "prod-secret"}); err != nil {
		t.Fatalf("failed to set server variables: %v", err)
	}

	pane.Request = &http.Request{Method: http.GET, URL: "/x", ServerID: &prod.ID, Headers: map[string]string{"X-Missing": "{{nope}}"}}

	resolved := pane.resolvedRequest()
	if resolved.Headers["X-Missing"] != "{{nope}}" {
		t.Errorf("expected an unknown placeholder to stay literal, got %+v", resolved.Headers)
	}
}

// TestRequestPane_ResolvedRequest_NoServerVariables_PlaceholderStaysLiteral
// covers an ad-hoc request with no configured server at all - there's no
// server to hold variable values, so a placeholder is left untouched
// rather than erroring out.
func TestRequestPane_ResolvedRequest_NoServerVariables_PlaceholderStaysLiteral(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.Request = &http.Request{Method: http.GET, URL: "https://example.com/{{id}}"}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://example.com/{{id}}" {
		t.Errorf("expected the placeholder to stay literal with no server, got %q", resolved.URL)
	}
}
