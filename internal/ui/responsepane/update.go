package responsepane

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// SaveResponseBodyMsg reports the outcome of saveResponseBodyCmd - a
// background file write (see handleSaveToFileKey), so ResponsePane can
// show a success/failure status line once it completes.
type SaveResponseBodyMsg struct {
	Path string
	Err  error
}

type ResponseCopiedMsg struct {
	Err error
}

// Update handles Bubble Tea messages and state transitions
func (m *ResponsePane) Update(msg tea.Msg) (*ResponsePane, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case gotoSequenceExpiredMsg:
		if m.gPending && msg.sequence == m.gotoSequence {
			m.gPending = false
			m.gotoLineBuffer = ""
		}
		return m, nil

	case SaveResponseBodyMsg:
		if msg.Err != nil {
			m.saveStatusMsg = fmt.Sprintf("failed to save: %v", msg.Err)
			m.saveStatusErr = true
		} else {
			m.saveStatusMsg = fmt.Sprintf("saved to %s", msg.Path)
			m.saveStatusErr = false
		}
		return m, nil

	case tea.KeyPressMsg:
		// The filename prompt (see SaveResponseRaw) claims every key while
		// open, same as groupspane's create/rename prompt - none of the
		// tab/scroll/copy bindings below should fire while typing a
		// filename (e.g. "y" in "my-response.json" must not also trigger
		// CopyResponse).
		if m.savingToFile {
			return m.handleSaveToFileKey(msg)
		}

		// The search prompt (see SearchResponse) claims every key while
		// open, same reason as the filename prompt above.
		if m.searching {
			return m.handleSearchInputKey(msg)
		}

		// The style picker (see PickBodyStyle) claims every key while
		// open, same reason as the filename prompt above.
		if m.pickingStyle {
			return m.handleStylePickerKey(msg)
		}

		// The saved-cookies/auth-headers viewer (see ViewAuthJar) claims
		// every key while open, same reason as the filename prompt above.
		if m.viewingJar {
			return m.handleJarViewerKey(msg)
		}

		// The past-sends viewer (see ViewHistory) claims every key while
		// open, same reason as the filename prompt above.
		if m.viewingHistory {
			return m.handleHistoryViewerKey(msg)
		}

		// Diff mode (see startDiff/"x" in history_viewer.go) isn't a
		// claims-every-key overlay the way the ones above are - scrolling
		// and search still work over the diff content exactly like they
		// do over a normal tab's (see updateViewportForActiveTab), since
		// both just feed the same viewport. Only esc (exit back to the
		// normal tabs) and a tab switch (which implies "I don't want to
		// see the diff anymore, show me that tab instead") need special
		// handling.
		if m.diffing && msg.String() == "esc" {
			m.exitDiff()
			return m, nil
		}
		if m.diffing && (keybindings.Matches(msg, m.keys.DirectTab) || keybindings.Matches(msg, m.keys.TabNavPrev) || keybindings.Matches(msg, m.keys.TabNavNext)) {
			m.diffing = false
		}

		// gg (go to top) / g{n}g (go to line n) - see navigation.go. Armed
		// by a bare "g" below; while armed, every key either continues the
		// sequence or cancels it and falls through to be handled normally.
		if m.gPending {
			if gotoCmd, consumed := m.continueGoto(msg.String()); consumed {
				return m, gotoCmd
			}
		}
		if keybindings.Matches(msg, m.keys.GotoTop) {
			return m, m.beginGoto()
		}
		if keybindings.Matches(msg, m.keys.GotoBottom) {
			m.viewport.GotoBottom()
			return m, nil
		}

		// "/" search, "n"/"b" cycle through its matches from wherever the
		// last search move left off, "N"/"P" instead from wherever the
		// view is currently scrolled to - see search.go.
		if keybindings.Matches(msg, m.keys.SearchResponse) {
			return m, m.openSearch()
		}
		if keybindings.Matches(msg, m.keys.SearchNext) {
			m.advanceSearchMatch(1)
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.SearchPrev) {
			m.advanceSearchMatch(-1)
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.SearchNextFromPosition) {
			m.jumpToNearestMatch(true)
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.SearchPrevFromPosition) {
			m.jumpToNearestMatch(false)
			return m, nil
		}

		// "t" opens the body syntax style picker - see style_picker.go.
		// Meaningless in load-test mode (nothing there ever goes through
		// chroma), same guard ToggleRawBody/SaveResponseRaw already use.
		if keybindings.Matches(msg, m.keys.PickBodyStyle) {
			if !m.isLoadTest {
				m.openStylePicker()
			}
			return m, nil
		}

		// "c" remembers this response's cookies/auth header for its host
		// (see jar_capture.go) regardless of the auto-remember settings;
		// "v" opens the viewer over whatever's currently remembered (see
		// jar_viewer.go). Both meaningless in load-test mode - there's no
		// single response/host to key by there.
		if keybindings.Matches(msg, m.keys.CaptureAuth) {
			if !m.isLoadTest {
				m.captureNow()
			}
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.ViewAuthJar) {
			if !m.isLoadTest {
				m.openJarViewer()
			}
			return m, nil
		}

		// "H" opens the past-sends viewer (see history_viewer.go) -
		// meaningless in load-test mode, same reasoning as CaptureAuth/
		// ViewAuthJar above (a load test's own many requests were never
		// recorded into history to begin with - see ResponsePane.
		// SetResponse, which SetLoadTestStats never calls).
		if keybindings.Matches(msg, m.keys.ViewHistory) {
			if !m.isLoadTest {
				m.openHistoryViewer()
			}
			return m, nil
		}

		// Direct tab access
		if keybindings.Matches(msg, m.keys.DirectTab) {
			switch msg.String() {
			case "1":
				m.activeTab = int(TabBody)
				m.updateViewportForActiveTab()
			case "2":
				m.activeTab = int(TabHeaders)
				m.updateViewportForActiveTab()
			case "3":
				m.activeTab = int(TabTiming)
				m.updateViewportForActiveTab()
			case "4":
				// Load test mode only has 3 tabs (Overview/Latency/Errors)
				// - there's no load-test equivalent of "the request that
				// was sent" to jump to (the JobConfig's request doesn't
				// change per stats update).
				if !m.isLoadTest {
					m.activeTab = int(TabRequest)
					m.updateViewportForActiveTab()
				}
			}
		}

		// Tab navigation
		if keybindings.Matches(msg, m.keys.TabNavPrev) {
			maxTabs := m.getMaxTabs()
			m.activeTab = (m.activeTab - 1 + maxTabs) % maxTabs
			m.updateViewportForActiveTab()
		}
		if keybindings.Matches(msg, m.keys.TabNavNext) {
			maxTabs := m.getMaxTabs()
			m.activeTab = (m.activeTab + 1) % maxTabs
			m.updateViewportForActiveTab()
		}

		// Copy handling
		if keybindings.Matches(msg, m.keys.CopyResponse) {
			if m.Response != nil && !m.isLoadTest {
				return m, m.copyToClipboard(m.Response.Body)
			}
		}

		// Raw/pretty toggle - only meaningful on the Body tab, which is
		// the only one formatContentByType touches at all.
		if keybindings.Matches(msg, m.keys.ToggleRawBody) {
			if m.Response != nil && !m.isLoadTest && TabIndex(m.activeTab) == TabBody {
				m.rawBody = !m.rawBody
				m.updateViewportForActiveTab()
			}
			return m, nil
		}

		// Wrap toggle - applies to whatever's currently displayed on any
		// tab (see setViewportContent), not just Body.
		if keybindings.Matches(msg, m.keys.ToggleWrap) {
			if !m.isLoadTest {
				m.wrapBody = !m.wrapBody
				m.updateViewportForActiveTab()
			}
			return m, nil
		}

		// Save-to-file prompt - always the raw response body (see
		// saveResponseBodyCmd), regardless of ToggleRawBody's current
		// display state or which tab happens to be active, same as
		// CopyResponse above.
		if keybindings.Matches(msg, m.keys.SaveResponseRaw) {
			if m.Response != nil && !m.isLoadTest {
				m.savingToFile = true
				m.saveStatusMsg = ""
				m.filenameInput.SetValue("")
				m.filenameInput.Focus()
				return m, textinput.Blink
			}
		}

		if keybindings.Matches(msg, m.keys.PageUp) {
			m.viewport.HalfPageUp()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.PageDown) {
			m.viewport.HalfPageDown()
			return m, nil
		}
	}

	// The style picker's list filters asynchronously - the "/" query typed
	// in handleStylePickerKey above only starts the match (bubbles' own
	// list.Model returns a tea.Cmd that resolves to a FilterMatchesMsg,
	// not a tea.KeyPressMsg), so that reply has to reach styleList.Update
	// here too, not just the key that triggered it - otherwise the query
	// text updates but the visible list never actually narrows.
	if m.pickingStyle {
		m.styleList, cmd = m.styleList.Update(msg)
		m.updateViewportForActiveTab()
		return m, cmd
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleSaveToFileKey handles input while the filename prompt is open
// (see SaveResponseRaw) - enter confirms and dispatches the save, esc
// cancels back to normal browsing, anything else is forwarded to
// filenameInput. Mirrors groupspane's create/rename prompt handling.
func (m *ResponsePane) handleSaveToFileKey(msg tea.KeyPressMsg) (*ResponsePane, tea.Cmd) {
	if msg.String() == "esc" {
		m.savingToFile = false
		m.filenameInput.Blur()
		return m, nil
	}
	if msg.String() == "enter" {
		path, err := resolveSavePath(m.filenameInput.Value())
		if err != nil {
			m.saveStatusMsg = err.Error()
			m.saveStatusErr = true
			return m, nil
		}
		m.savingToFile = false
		m.filenameInput.Blur()
		return m, saveResponseBodyCmd(path, m.Response.Body)
	}

	var cmd tea.Cmd
	m.filenameInput, cmd = m.filenameInput.Update(msg)
	return m, cmd
}

// resolveSavePath validates and expands the user-typed filename: a
// leading "~" is expanded to the home directory (bare os.WriteFile doesn't
// do this on its own - a raw "~/x" would otherwise create a literal
// directory named "~"), and a blank filename is rejected outright rather
// than silently doing nothing. Anything else (relative or absolute) is
// passed straight through to os.WriteFile, same as internal/cli/output.go
// already does for its own -output flag.
func resolveSavePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}
	if raw == "~" || strings.HasPrefix(raw, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve ~: %w", err)
		}
		raw = filepath.Join(home, strings.TrimPrefix(raw, "~"))
	}
	return raw, nil
}

// saveResponseBodyCmd writes content (always the raw response body, never
// the formatted/highlighted view - see ToggleRawBody) to path in the
// background, off the UI goroutine, reporting the outcome via
// SaveResponseBodyMsg.
func saveResponseBodyCmd(path, content string) tea.Cmd {
	return func() tea.Msg {
		err := os.WriteFile(path, []byte(content), 0644)
		return SaveResponseBodyMsg{Path: path, Err: err}
	}
}

// copyToClipboard copies content to the system clipboard
func (m ResponsePane) copyToClipboard(content string) tea.Cmd {
	return copyCommand(content, clipboard.WriteAll)
}

func copyCommand(content string, write func(string) error) tea.Cmd {
	return func() tea.Msg {
		if write == nil {
			return ResponseCopiedMsg{Err: errors.New("clipboard writer is unavailable")}
		}
		return ResponseCopiedMsg{Err: write(content)}
	}
}

// updateViewportForActiveTab updates the viewport content based on the active tab
func (m *ResponsePane) updateViewportForActiveTab() {
	if m.diffing {
		m.setViewportContent(m.diffContent)
		return
	}

	if m.isLoadTest {
		m.updateLoadTestTabContent()
		return
	}

	if m.Response == nil {
		return
	}

	var content string
	switch TabIndex(m.activeTab) {
	case TabBody:
		switch {
		case m.Response.Error != "":
			content = m.Response.Error
		case m.rawBody:
			content = highlightRawBody(m.Response.Body, m.Response.ParseContentType(), m.effectiveBodyStyle())
		default:
			contentType := m.Response.ParseContentType()
			content = formatContentByType(m.Response.Body, contentType, m.effectiveBodyStyle())
		}
	case TabHeaders:
		content = m.renderHeaders()
	case TabTiming:
		content = m.renderTiming()
	case TabRequest:
		content = m.renderRequest()
	}
	m.setViewportContent(content)
}

// updateLoadTestTabContent updates the viewport content for load test tabs
func (m *ResponsePane) updateLoadTestTabContent() {
	if m.LoadTestStats == nil {
		m.setViewportContent("No data")
		return
	}

	var content string
	switch TabIndex(m.activeTab) {
	case TabLoadTestOverview:
		content = m.renderLoadTestOverview()
	case TabLoadTestLatency:
		content = m.renderLoadTestLatency()
	case TabLoadTestErrors:
		content = m.renderLoadTestErrors()
	}
	m.setViewportContent(content)
}
