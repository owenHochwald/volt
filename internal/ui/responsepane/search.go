package responsepane

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// searchSnapshot captures everything a search session (see openSearch)
// needs to fully undo - restored verbatim by cancelSearch, the same way
// esc cancelling any other prompt leaves the pane exactly as it was.
type searchSnapshot struct {
	offset    int
	query     string
	matches   []searchMatch
	matchIdx  int
	statusMsg string
}

// searchMatch is a single occurrence of the search query - a specific
// span within a specific line, not just "the query appears somewhere on
// this line" - so a line where the query occurs more than once
// contributes one searchMatch per occurrence, each independently
// counted, jumped to, and highlighted (see findSearchMatches/
// highlightMatches). start/end are byte offsets into that line's
// ansi.Strip'd plain text (what the search regex actually matches
// against - see findSearchMatches), not the original ANSI-styled line.
type searchMatch struct {
	line       int
	start, end int
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
	if m.wrapBody && m.viewport.Width() > 0 {
		content = wrapLines(content, m.viewport.Width())
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
		offset:    m.viewport.YOffset(),
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
func (m *ResponsePane) handleSearchInputKey(msg tea.KeyPressMsg) (*ResponsePane, tea.Cmd) {
	if msg.String() == "esc" {
		m.cancelSearch()
		return m, nil
	}
	if msg.String() == "enter" {
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

	matches := findSearchMatches(m.currentContent, query)
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
	for i, match := range matches {
		if match.line >= m.searchOrigin.offset {
			idx = i
			break
		}
	}
	if idx == -1 {
		idx = 0
	}
	m.searchMatchIdx = idx
	m.scrollToLine(matches[idx].line)
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
	m.scrollToLine(m.searchMatches[m.searchMatchIdx].line)
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

	if m.searchMatchIdx >= 0 && m.searchMatchIdx < len(m.searchMatches) && m.isLineVisible(m.searchMatches[m.searchMatchIdx].line) {
		delta := 1
		if !forward {
			delta = -1
		}
		m.advanceSearchMatch(delta)
		return
	}

	pos := m.viewport.YOffset()
	idx := -1
	if forward {
		for i, match := range m.searchMatches {
			if match.line >= pos {
				idx = i
				break
			}
		}
		if idx == -1 {
			idx = 0
		}
	} else {
		for i := len(m.searchMatches) - 1; i >= 0; i-- {
			if m.searchMatches[i].line <= pos {
				idx = i
				break
			}
		}
		if idx == -1 {
			idx = len(m.searchMatches) - 1
		}
	}
	m.searchMatchIdx = idx
	m.scrollToLine(m.searchMatches[idx].line)
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

// findSearchMatches returns every occurrence of query (see
// compileSearchQuery) in content (split on "\n"), in ascending order - a
// line where the query occurs more than once contributes one searchMatch
// per occurrence, not one for the whole line, so each is counted, jumped
// to, and highlighted independently (see highlightMatches). content may
// carry ANSI escape codes (chroma's syntax highlighting in the pretty
// body view - see formatContentByType) wrapping individual tokens, which
// matching against the raw string would otherwise silently miss whenever
// a match's characters fall under different color spans - stripping them
// first (into plain, matched against instead of line itself) is what
// makes search work the same in pretty view as it does in raw. The
// resulting byte offsets are still only valid against that same plain
// text - see highlightMatches, which re-derives it the same way to stay
// in sync.
func findSearchMatches(content, query string) []searchMatch {
	if query == "" {
		return nil
	}
	re := compileSearchQuery(query)
	lines := strings.Split(content, "\n")
	var matches []searchMatch
	for i, line := range lines {
		plain := ansi.Strip(line)
		for _, span := range re.FindAllStringIndex(plain, -1) {
			matches = append(matches, searchMatch{line: i, start: span[0], end: span[1]})
		}
	}
	return matches
}

// highlightMatches re-renders content with just the matched query text
// itself highlighted - the current match (matches[currentIdx]) distinctly
// from the rest, so it's obvious which one "n"/"b" will move away from,
// even when it shares a line with other matches. Delegates the actual
// per-line rendering to highlightLineSpans, which preserves that line's
// own syntax coloring outside the highlighted spans - only the matched
// text itself trades its original color for style's flat one.
func highlightMatches(content, query string, matches []searchMatch, currentIdx int) string {
	if len(matches) == 0 || query == "" {
		return content
	}
	byLine := make(map[int][]styledSpan, len(matches))
	for i, match := range matches {
		style := otherMatchStyle
		if i == currentIdx {
			style = currentMatchStyle
		}
		byLine[match.line] = append(byLine[match.line], styledSpan{match.start, match.end, style})
	}

	lines := strings.Split(content, "\n")
	for lineNum, spans := range byLine {
		lines[lineNum] = highlightLineSpans(lines[lineNum], spans)
	}
	return strings.Join(lines, "\n")
}

// styledSpan is one highlighted region within a single line - start/end
// are byte offsets into that line's ansi.Strip'd plain text (matching
// searchMatch), rendered in style (see highlightLineSpans).
type styledSpan struct {
	start, end int
	style      lipgloss.Style
}

// highlightLineSpans re-renders original with each span in spans (already
// in ascending order - see findSearchMatches, which scans left to right
// within a line) wrapped in its own style, leaving the rest of the line
// as-is. Spans are byte offsets into ansi.Strip(original), translated
// here to cell-column offsets (ansi.Cut/StringWidth operate in display
// cells, not bytes, so a multi-byte character earlier in the line doesn't
// throw off where a later highlight actually lands).
//
// A matched span is rendered from its *plain* text (plain[start:end]),
// not ansi.Cut of the original - fine-grained syntax highlighting (e.g.
// chroma emitting a separate color+reset around nearly every token, as it
// does for JSON) can leave an ANSI-reconstructed cut carrying an embedded
// reset of its own; wrapping that in span.style's own SGR codes would
// have that inner reset cut the wrapping style right back off partway
// through, before any of the actual characters. Plain text has no such
// embedded codes to fight with. The gaps *between*/around spans don't
// have this problem (nothing wraps them) and are still cut straight from
// the original ANSI-styled line - ansi.Cut reopens whatever styling
// already applies to that range on its own - preserving the line's
// existing syntax coloring everywhere except the highlighted spans
// themselves, rather than discarding it for the whole line the way
// rendering from ansi.Strip'd text alone would.
func highlightLineSpans(original string, spans []styledSpan) string {
	if len(spans) == 0 {
		return original
	}
	plain := ansi.Strip(original)

	// The gaps between/around spans need whatever style was active in
	// original at that point reapplied, in case a span falls in the
	// middle of a longer styled run (e.g. a whole quoted string as one
	// chroma token) - ansi.Cut does exactly this, but by re-scanning
	// original from the start on every call. Calling it once per gap
	// made a line with many matches quadratic (matches * line length) -
	// splitOriginalByPlainOffsets computes every gap's cut in a single
	// linear pass instead.
	offsets := make([]int, 0, len(spans)*2+2)
	offsets = append(offsets, 0)
	for _, span := range spans {
		offsets = append(offsets, span.start, span.end)
	}
	offsets = append(offsets, len(plain))
	gaps := splitOriginalByPlainOffsets(original, offsets)

	var b strings.Builder
	for i, span := range spans {
		b.WriteString(gaps[2*i])
		if span.end > span.start {
			b.WriteString(span.style.Render(plain[span.start:span.end]))
		}
	}
	b.WriteString(gaps[len(gaps)-1])
	return b.String()
}

// splitOriginalByPlainOffsets returns the len(offsets)-1 substrings of
// original corresponding to each [offsets[i], offsets[i+1]) range of
// *plain* (ansi.Strip'd) byte offsets (offsets must be sorted and start
// at 0), each with whatever SGR style was active at its start reapplied
// as a prefix - the same reconstruction ansi.Cut does for a single range,
// computed here for every requested offset in one linear pass over
// original instead. Only tracks simple "ESC [ ... m" (SGR) sequences,
// which is all chroma's syntax highlighting and this package's own
// search-highlight styles ever emit; any other escape sequence is passed
// through as part of whichever segment it falls in without being
// specially tracked.
func splitOriginalByPlainOffsets(original string, offsets []int) []string {
	segments := make([]string, 0, max(len(offsets)-1, 0))
	if len(offsets) < 2 {
		return segments
	}

	var activeStyle strings.Builder
	var seg strings.Builder
	plainPos := 0
	nextCut := 1

	flush := func() {
		segments = append(segments, activeStyle.String()+seg.String())
		seg.Reset()
	}

	i, n := 0, len(original)
	for i < n {
		if original[i] == 0x1b && i+1 < n && original[i+1] == '[' {
			j := i + 2
			for j < n && original[j] != 'm' && (original[j] < 'A' || original[j] > 'Z') && (original[j] < 'a' || original[j] > 'z') {
				j++
			}
			if j < n {
				j++ // include the terminator byte (usually 'm')
			}
			code := original[i:j]
			seg.WriteString(code)
			if code == "\x1b[0m" || code == "\x1b[m" {
				activeStyle.Reset()
			} else if strings.HasSuffix(code, "m") {
				activeStyle.WriteString(code)
			}
			i = j
			continue
		}
		_, size := utf8.DecodeRuneInString(original[i:])
		if size == 0 {
			size = 1
		}
		seg.WriteString(original[i : i+size])
		plainPos += size
		i += size
		for nextCut < len(offsets) && plainPos >= offsets[nextCut] {
			flush()
			nextCut++
		}
	}
	for nextCut < len(offsets) {
		flush()
		nextCut++
	}
	return segments
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
