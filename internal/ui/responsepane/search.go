package responsepane

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// searchSnapshot captures everything a search session (see openSearch)
// needs to fully undo - restored verbatim by cancelSearch, the same way
// esc cancelling any other prompt leaves the pane exactly as it was.
type searchSnapshot struct {
	offset    int
	query     string
	matches   []int
	matchIdx  int
	statusMsg string
}

// setViewportContent replaces the viewport's content, keeping currentContent
// (see ResponsePane) in sync so search always has the raw string to match
// against - and resetting any active search whenever the content actually
// changes (a new response, a different tab, or the raw/pretty toggle - see
// ToggleRawBody), since matches are line indices into content and become
// meaningless once it's replaced by something else. Left alone when
// content is unchanged (e.g. switching to another tab and back) so
// browsing a search's matches isn't lost by merely looking elsewhere.
// Always (re)renders through highlightMatches so the viewport shows
// whatever the active search (if any) has found, not just the plain
// content.
func (m *ResponsePane) setViewportContent(content string) {
	if content != m.currentContent {
		m.resetSearch()
	}
	m.currentContent = content
	m.refreshHighlightedContent()
}

// refreshHighlightedContent re-renders currentContent into the viewport
// with the active search's matches highlighted (see highlightMatches) -
// called whenever the search state changes (typing, cancelling, n/b)
// without the underlying content itself changing, so highlighting always
// reflects the current query/position without going through
// setViewportContent's own "content changed" invalidation. Also applies
// ToggleWrap's wrapBody setting (see wrapLines) - after highlighting, not
// before, so search's own line numbering (searchMatches) stays in terms
// of currentContent's actual lines; a long line wrapping into several
// visual ones can therefore make a jump-to-match land a little off
// afterward, an accepted tradeoff for readable wrapped text.
func (m *ResponsePane) refreshHighlightedContent() {
	content := highlightMatches(m.currentContent, m.searchQuery, m.searchMatches, m.searchMatchIdx)
	if m.wrapBody && m.viewport.Width > 0 {
		content = wrapLines(content, m.viewport.Width)
	}
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
// rendered inline in place of the footer hint (see renderNormalView) -
// snapshotting the current scroll position and any already-confirmed
// search so cancelSearch can undo this session's typing back to exactly
// this state.
func (m *ResponsePane) openSearch() tea.Cmd {
	m.searchOrigin = searchSnapshot{
		offset:    m.viewport.YOffset,
		query:     m.searchQuery,
		matches:   m.searchMatches,
		matchIdx:  m.searchMatchIdx,
		statusMsg: m.searchStatusMsg,
	}
	m.searching = true
	m.searchInput.SetValue("")
	m.searchInput.Focus()
	return textinput.Blink
}

// handleSearchInputKey handles input while the search prompt is open -
// esc cancels back to exactly how things were before this search session
// opened (see cancelSearch), enter simply stops editing and keeps
// whatever the live search already landed on (see liveSearch, run after
// every other keystroke), anything else is forwarded to searchInput.
func (m *ResponsePane) handleSearchInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.cancelSearch()
		return m, nil
	}
	if msg.String() == tea.KeyEnter.String() {
		m.searching = false
		m.searchInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.liveSearch()
	return m, cmd
}

// cancelSearch restores everything openSearch snapshotted - scroll
// position, and whatever search (if any) was already active before this
// session - discarding this session's typing entirely, the same as esc
// cancelling the save-to-file prompt discards that filename without
// touching the response itself.
func (m *ResponsePane) cancelSearch() {
	m.searching = false
	m.searchInput.Blur()
	m.searchQuery = m.searchOrigin.query
	m.searchMatches = m.searchOrigin.matches
	m.searchMatchIdx = m.searchOrigin.matchIdx
	m.searchStatusMsg = m.searchOrigin.statusMsg
	m.viewport.SetYOffset(m.searchOrigin.offset)
	m.refreshHighlightedContent()
}

// liveSearch re-runs the search on every keystroke (vim/less-style
// incremental search - see handleSearchInputKey) against currentContent
// (whatever's currently displayed - raw or pretty body, headers, timing,
// or the sent request), always relative to searchOrigin.offset (the
// scroll position when "/" was pressed) rather than wherever the live
// preview has since scrolled to - so backspacing back to a shorter query
// re-searches from the same starting point instead of drifting further
// down with every keystroke. Finds the first match at or after that
// position, wrapping to the very first match if none qualify - the
// forward, wrap-around search vim/less's "/" does.
func (m *ResponsePane) liveSearch() {
	query := strings.TrimSpace(m.searchInput.Value())
	if query == "" {
		m.searchQuery = ""
		m.searchMatches = nil
		m.searchMatchIdx = -1
		m.searchStatusMsg = ""
		m.viewport.SetYOffset(m.searchOrigin.offset)
		m.refreshHighlightedContent()
		return
	}

	matches := searchMatchLines(m.currentContent, query)
	m.searchQuery = query
	m.searchMatches = matches

	if len(matches) == 0 {
		m.searchMatchIdx = -1
		m.searchStatusMsg = fmt.Sprintf("no matches for %q", query)
		m.refreshHighlightedContent()
		return
	}

	m.searchStatusMsg = ""
	idx := -1
	for i, line := range matches {
		if line >= m.searchOrigin.offset {
			idx = i
			break
		}
	}
	if idx == -1 {
		idx = 0
	}
	m.searchMatchIdx = idx
	m.scrollToLine(matches[idx])
	m.refreshHighlightedContent()
}

// advanceSearchMatch moves to the next (delta=1) or previous (delta=-1)
// match, wrapping around - bound to SearchNext ("n") / SearchPrev ("b").
// Always continues from searchMatchIdx, the remembered position in the
// match sequence - wherever the *last search move* left off, regardless
// of where the view has since scrolled to on its own (see
// jumpToNearestMatch's "N"/"P" for that). A no-op when there's no active
// search.
func (m *ResponsePane) advanceSearchMatch(delta int) {
	if len(m.searchMatches) == 0 {
		return
	}
	m.searchMatchIdx = (m.searchMatchIdx + delta + len(m.searchMatches)) % len(m.searchMatches)
	m.scrollToLine(m.searchMatches[m.searchMatchIdx])
	m.refreshHighlightedContent()
}

// jumpToNearestMatch is SearchNextFromPosition/SearchPrevFromPosition's
// ("N"/"P") counterpart to advanceSearchMatch ("n"/"b"): if the
// remembered match (searchMatchIdx) is still visible on screen - an
// ordinary scroll (j/k, pgup/pgdn) that never actually scrolled it out of
// view - it behaves exactly like "n"/"b", just continuing from there.
// Only once it's no longer visible (a real jump elsewhere - gg/G/{n}g)
// does it re-anchor to wherever the viewport is *currently* scrolled to
// instead of picking up wherever that earlier search left off.
// forward=true searches down from the current position (wrapping to the
// first match if none qualify below it); forward=false searches up
// (wrapping to the last match). A no-op when there's no active search.
func (m *ResponsePane) jumpToNearestMatch(forward bool) {
	if len(m.searchMatches) == 0 {
		return
	}

	if m.searchMatchIdx >= 0 && m.searchMatchIdx < len(m.searchMatches) && m.isLineVisible(m.searchMatches[m.searchMatchIdx]) {
		delta := 1
		if !forward {
			delta = -1
		}
		m.advanceSearchMatch(delta)
		return
	}

	pos := m.viewport.YOffset
	idx := -1
	if forward {
		for i, line := range m.searchMatches {
			if line >= pos {
				idx = i
				break
			}
		}
		if idx == -1 {
			idx = 0
		}
	} else {
		for i := len(m.searchMatches) - 1; i >= 0; i-- {
			if m.searchMatches[i] <= pos {
				idx = i
				break
			}
		}
		if idx == -1 {
			idx = len(m.searchMatches) - 1
		}
	}
	m.searchMatchIdx = idx
	m.scrollToLine(m.searchMatches[idx])
	m.refreshHighlightedContent()
}

// compileSearchQuery compiles query as a case-insensitive regular
// expression (Go's RE2 syntax - "." "?" "*" "+", character classes,
// alternation, anchors, and so on all work), so the "/" search prompt
// doubles as a regex search whenever the query happens to parse as one.
// A query that *isn't* valid regex syntax - unbalanced parens/brackets are
// common in real body/header text - falls back to matching it as a plain
// literal string instead of rejecting the search outright; regexp.
// QuoteMeta escapes every metacharacter, so this fallback always compiles.
// RE2 (unlike PCRE-style backtracking engines) guarantees linear-time
// matching, so an adversarial or accidentally-catastrophic pattern typed
// here can't hang the UI.
func compileSearchQuery(query string) *regexp.Regexp {
	if re, err := regexp.Compile("(?i)" + query); err == nil {
		return re
	}
	return regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
}

// searchMatchLines returns the 0-based line indices of content (split on
// "\n") that match query (see compileSearchQuery). content may carry ANSI
// escape codes (chroma's syntax highlighting in the pretty body view -
// see formatContentByType) wrapping individual tokens, which matching
// against the raw string would otherwise silently miss whenever a match's
// characters fall under different color spans - stripping them first is
// what makes search work the same in pretty view as it does in raw.
func searchMatchLines(content, query string) []int {
	if query == "" {
		return nil
	}
	re := compileSearchQuery(query)
	lines := strings.Split(content, "\n")
	var matches []int
	for i, line := range lines {
		if re.MatchString(ansi.Strip(line)) {
			matches = append(matches, i)
		}
	}
	return matches
}

// highlightMatches re-renders content with just the matched query text
// itself highlighted on every matching line - the current match
// (matches[currentIdx]) distinctly from the rest, so it's obvious which
// one "n"/"b" will move away from. Rather than trying to layer a
// highlight on top of a line's existing ANSI styling (chroma's syntax
// highlighting embeds its own reset codes throughout, which would keep
// cutting an outer background right back off), a matching line is
// re-rendered from its plain (ansi.Strip'd) text - trading that line's
// syntax coloring for a reliable highlight instead. Only the query's own
// characters get the background, not the rest of the line around them.
func highlightMatches(content, query string, matches []int, currentIdx int) string {
	if len(matches) == 0 || query == "" {
		return content
	}
	re := compileSearchQuery(query)
	currentLine := -1
	if currentIdx >= 0 && currentIdx < len(matches) {
		currentLine = matches[currentIdx]
	}
	matchSet := make(map[int]bool, len(matches))
	for _, line := range matches {
		matchSet[line] = true
	}

	lines := strings.Split(content, "\n")
	for i := range lines {
		if !matchSet[i] {
			continue
		}
		plain := ansi.Strip(lines[i])
		style := otherMatchStyle
		if i == currentLine {
			style = currentMatchStyle
		}
		lines[i] = highlightQueryInLine(plain, re, style)
	}
	return strings.Join(lines, "\n")
}

// highlightQueryInLine wraps every non-overlapping match of re within line
// in style, leaving the rest of the line as plain text. A pattern capable
// of matching an empty string (e.g. "a*") can produce zero-length spans
// between characters - FindAllStringIndex still terminates (it advances by
// a rune whenever a match is empty), it just renders as a run of empty
// highlighted spans rather than anything meaningful; the same oddity any
// regex tool shows for such a pattern, not a bug specific to this search.
func highlightQueryInLine(line string, re *regexp.Regexp, style lipgloss.Style) string {
	spans := re.FindAllStringIndex(line, -1)
	if len(spans) == 0 {
		return line
	}
	var b strings.Builder
	last := 0
	for _, span := range spans {
		b.WriteString(line[last:span[0]])
		b.WriteString(style.Render(line[span[0]:span[1]]))
		last = span[1]
	}
	b.WriteString(line[last:])
	return b.String()
}

// wrapLines wraps content to width columns (see ToggleWrap/wrapBody),
// ANSI-aware (ansi.Wrap preserves embedded styling - search highlights,
// syntax colors - across the wrap, and already treats content's own
// "\n"s as hard breaks, so it's safe to call on the whole multi-line
// string directly rather than line by line). Unlike a fixed-size preview
// box, the viewport this feeds is already scrollable, so even a single
// pathologically long line (a minified page's whole body on one line,
// say) wrapping into many visual ones is just more to scroll through,
// not a layout hazard.
func wrapLines(content string, width int) string {
	return ansi.Wrap(content, width, "")
}
