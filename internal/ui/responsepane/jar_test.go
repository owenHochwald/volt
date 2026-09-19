package responsepane

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/authjar"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func setupPaneWithJarAndResponse(t *testing.T, jar *authjar.Store, sentHeaders map[string]string) *ResponsePane {
	t.Helper()
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetWidth(120)
	pane.SetHeight(30)
	pane.SetJarViewerSize(60, 20)
	pane.SetJar(jar)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
			"Set-Cookie":   {"session=abc123; Path=/", "csrftoken=xyz789; Path=/"},
		},
		Body: `{"ok": true}`,
	}, &http.Request{Method: http.GET, URL: "http://api.example.com/login", Headers: sentHeaders})
	return &pane
}

func TestResponsePane_CaptureAuth_RemembersCookiesAndAuthHeader(t *testing.T) {
	jar := authjar.NewStore()
	pane := setupPaneWithJarAndResponse(t, jar, map[string]string{"Authorization": "Bearer token123"})

	model, _ := pane.Update(keyRunes("c"))
	_ = model

	headers := jar.HeadersFor("api.example.com")
	if headers["Authorization"] != "Bearer token123" {
		t.Errorf("Authorization = %q, want %q", headers["Authorization"], "Bearer token123")
	}
	if !strings.Contains(headers["Cookie"], "session=abc123") || !strings.Contains(headers["Cookie"], "csrftoken=xyz789") {
		t.Errorf("Cookie = %q, want both session and csrftoken", headers["Cookie"])
	}
}

func TestResponsePane_CaptureAuth_IgnoresCaseOfHeaderName(t *testing.T) {
	jar := authjar.NewStore()
	pane := setupPaneWithJarAndResponse(t, jar, map[string]string{"authorization": "Basic xyz"})

	_, _ = pane.Update(keyRunes("c"))

	if got := jar.HeadersFor("api.example.com")["Authorization"]; got != "Basic xyz" {
		t.Errorf("Authorization = %q, want %q", got, "Basic xyz")
	}
}

func TestResponsePane_CaptureAuth_NoAuthorizationHeaderSent_OnlyRemembersCookies(t *testing.T) {
	jar := authjar.NewStore()
	pane := setupPaneWithJarAndResponse(t, jar, map[string]string{})

	_, _ = pane.Update(keyRunes("c"))

	headers := jar.HeadersFor("api.example.com")
	if _, ok := headers["Authorization"]; ok {
		t.Error("expected no Authorization to be remembered when the request sent none")
	}
	if headers["Cookie"] == "" {
		t.Error("expected cookies to still be remembered")
	}
}

// TestResponsePane_AutoRememberCookies_CapturesOnArrivalWhenEnabled is a
// regression test for the "automatic" half of the feature: with the
// setting on, SetResponse itself must capture - the user shouldn't have
// to press "c" at all.
func TestResponsePane_AutoRememberCookies_CapturesOnArrivalWhenEnabled(t *testing.T) {
	jar := authjar.NewStore()
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetJar(jar)
	pane.SetAutoRememberCookies(true)

	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Set-Cookie": {"session=auto1"}},
	}, &http.Request{Method: http.GET, URL: "http://api.example.com/x"})

	if got := jar.HeadersFor("api.example.com")["Cookie"]; got != "session=auto1" {
		t.Errorf("Cookie = %q, want %q", got, "session=auto1")
	}
}

func TestResponsePane_AutoRememberCookies_Disabled_DoesNotCaptureAutomatically(t *testing.T) {
	jar := authjar.NewStore()
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetJar(jar)
	// Deliberately not calling SetAutoRememberCookies - false is the
	// default (see design.AppConfig's own field comment).

	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Set-Cookie": {"session=auto1"}},
	}, &http.Request{Method: http.GET, URL: "http://api.example.com/x"})

	if got := jar.HeadersFor("api.example.com"); got != nil {
		t.Errorf("expected nothing captured automatically while disabled, got %v", got)
	}
}

func TestResponsePane_AutoRememberAuthHeaders_IndependentOfCookies(t *testing.T) {
	jar := authjar.NewStore()
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetJar(jar)
	pane.SetAutoRememberAuthHeaders(true)
	// Cookies auto-remember left off.

	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Set-Cookie": {"session=auto1"}},
	}, &http.Request{Method: http.GET, URL: "http://api.example.com/x", Headers: map[string]string{"Authorization": "Bearer auto"}})

	headers := jar.HeadersFor("api.example.com")
	if headers["Authorization"] != "Bearer auto" {
		t.Errorf("Authorization = %q, want %q", headers["Authorization"], "Bearer auto")
	}
	if _, ok := headers["Cookie"]; ok {
		t.Error("expected cookies NOT to be auto-captured while that setting is off")
	}
}

func TestResponsePane_ViewAuthJar_OpensAndShowsSavedEntries(t *testing.T) {
	jar := authjar.NewStore()
	jar.RememberAuthHeader("api.example.com", "Authorization", "Bearer token123")
	pane := setupPaneWithJarAndResponse(t, jar, nil)

	model, _ := pane.Update(keyRunes("v"))
	result := model
	if !result.IsViewingJar() {
		t.Fatal("expected \"v\" to open the jar viewer")
	}

	view := result.JarViewerListView()
	if !strings.Contains(view, "Authorization") {
		t.Errorf("expected the saved Authorization header to be listed, got:\n%s", view)
	}
}

func TestResponsePane_ViewAuthJar_DeleteRemovesSelectedEntry(t *testing.T) {
	jar := authjar.NewStore()
	jar.RememberAuthHeader("api.example.com", "Authorization", "Bearer token123")
	pane := setupPaneWithJarAndResponse(t, jar, nil)

	model, _ := pane.Update(keyRunes("v"))
	opened := model

	model, _ = opened.Update(keyRunes("d"))
	result := model

	if got := jar.HeadersFor("api.example.com"); got != nil {
		t.Errorf("expected the Authorization header to be deleted, got %v", got)
	}
	if strings.Contains(result.JarViewerListView(), "Authorization") {
		t.Error("expected the deleted entry to no longer be listed")
	}
}

func TestResponsePane_ViewAuthJar_ClearAllRemovesEverythingForHost(t *testing.T) {
	jar := authjar.NewStore()
	jar.RememberAuthHeader("api.example.com", "Authorization", "Bearer token123")
	jar.RememberCookies("api.example.com", nil)
	jar.RememberAuthHeader("api.example.com", "X-Api-Key", "secret")
	pane := setupPaneWithJarAndResponse(t, jar, nil)

	model, _ := pane.Update(keyRunes("v"))
	opened := model

	model, _ = opened.Update(tea.KeyPressMsg(tea.Key{Text: "D"}))
	_ = model

	if got := jar.HeadersFor("api.example.com"); got != nil {
		t.Errorf("expected everything cleared for the host, got %v", got)
	}
}

func TestResponsePane_ViewAuthJar_EscClosesViewer(t *testing.T) {
	jar := authjar.NewStore()
	pane := setupPaneWithJarAndResponse(t, jar, nil)

	model, _ := pane.Update(keyRunes("v"))
	opened := model

	model, _ = opened.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	result := model
	if result.IsViewingJar() {
		t.Error("expected esc to close the jar viewer")
	}
}

// TestResponsePane_JarIndicator_ShowsOnlyWhenSomethingIsRemembered is a
// regression test for the response pane's status-bar hint - it must stay
// silent for a host with nothing remembered, and appear once something
// is.
func TestResponsePane_JarIndicator_ShowsOnlyWhenSomethingIsRemembered(t *testing.T) {
	jar := authjar.NewStore()
	pane := setupPaneWithJarAndResponse(t, jar, nil)

	if got := pane.jarIndicator(); got != "" {
		t.Errorf("expected no indicator before anything is remembered, got %q", got)
	}

	jar.RememberAuthHeader("api.example.com", "Authorization", "Bearer token123")
	if got := pane.jarIndicator(); got == "" {
		t.Error("expected an indicator once something is remembered for the host")
	}
}
