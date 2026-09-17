package responsepane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// setViewportContent replaces the viewport's content, keeping currentContent
// (see ResponsePane) in sync so search always has the raw string to match
// against - and resetting any active search whenever the content actually
// changes (a new response, a different tab, or the raw/pretty toggle - see
// ToggleRawBody), since matches are line indices into content and become
// meaningless once it's replaced by something else. Left alone when
// content is unchanged (e.g. switching to another tab and back) so
// browsing a search's matches isn't lost by merely looking elsewhere.
func (m *ResponsePane) setViewportContent(content string) {
	if content != m.currentContent {
		m.resetSearch()
	}
	m.currentContent = content
	m.viewport.SetContent(content)
}

// resetSearch clears any active search - the query, its matches, and any
// "no matches" status - back to "nothing searched".
func (m *ResponsePane) resetSearch() {
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchMatchIdx = -1
	m.searchStatusMsg = ""
}

// openSearch opens the "/" search prompt (see ResponsePane.searching),
// mirroring how SaveResponseRaw opens the filename prompt.
func (m *ResponsePane) openSearch() tea.Cmd {
	m.searching = true
	m.searchStatusMsg = ""
	m.searchInput.SetValue("")
	m.searchInput.Focus()
	return textinput.Blink
}

// handleSearchInputKey handles input while the search prompt is open -
// enter confirms (see confirmSearch), esc cancels back to normal
// browsing (leaving any previously confirmed search's matches intact, the
// same as esc cancelling the save-to-file prompt doesn't discard the
// response itself), anything else is forwarded to searchInput.
func (m *ResponsePane) handleSearchInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.searching = false
		m.searchInput.Blur()
		return m, nil
	}
	if msg.String() == tea.KeyEnter.String() {
		m.confirmSearch()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// confirmSearch runs the typed query against currentContent (whatever's
// currently displayed - raw or pretty body, headers, timing, or the sent
// request - see setViewportContent) and, if it finds anything, jumps to
// the first match at or after the current scroll position (wrapping to
// the very first match otherwise) - mirroring vim/less's forward "/"
// search. An empty query clears the search instead of running it, same
// as typing nothing and pressing enter in most search prompts.
func (m *ResponsePane) confirmSearch() {
	query := strings.TrimSpace(m.searchInput.Value())
	m.searching = false
	m.searchInput.Blur()

	if query == "" {
		m.resetSearch()
		return
	}

	matches := searchMatchLines(m.currentContent, query)
	m.searchQuery = query
	m.searchMatches = matches

	if len(matches) == 0 {
		m.searchMatchIdx = -1
		m.searchStatusMsg = fmt.Sprintf("no matches for %q", query)
		return
	}

	m.searchStatusMsg = ""
	start := m.viewport.YOffset
	idx := -1
	for i, line := range matches {
		if line >= start {
			idx = i
			break
		}
	}
	if idx == -1 {
		idx = 0
	}
	m.searchMatchIdx = idx
	m.scrollToLine(matches[idx])
}

// advanceSearchMatch moves to the next (delta=1) or previous (delta=-1)
// match, wrapping around - bound to SearchNext ("n") / SearchPrev ("b").
// A no-op when there's no active search.
func (m *ResponsePane) advanceSearchMatch(delta int) {
	if len(m.searchMatches) == 0 {
		return
	}
	m.searchMatchIdx = (m.searchMatchIdx + delta + len(m.searchMatches)) % len(m.searchMatches)
	m.scrollToLine(m.searchMatches[m.searchMatchIdx])
}

// searchMatchLines returns the 0-based line indices of content (split on
// "\n") that contain query, case-insensitively. content may carry ANSI
// escape codes (chroma's syntax highlighting in the pretty body view -
// see formatContentByType) wrapping individual tokens, which a plain
// substring search would otherwise silently miss whenever a match's
// characters fall under different color spans - stripping them first is
// what makes search work the same in pretty view as it does in raw.
func searchMatchLines(content, query string) []int {
	if query == "" {
		return nil
	}
	q := strings.ToLower(query)
	lines := strings.Split(content, "\n")
	var matches []int
	for i, line := range lines {
		if strings.Contains(strings.ToLower(ansi.Strip(line)), q) {
			matches = append(matches, i)
		}
	}
	return matches
}
