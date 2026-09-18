package responsepane

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func typeAndConfirmSearch(t *testing.T, pane *ResponsePane, query string) *ResponsePane {
	t.Helper()
	model, _ := pane.Update(keyRunes("/"))
	result := mustResponsePane(t, model)
	if !result.searching {
		t.Fatal("expected \"/\" to open the search prompt")
	}

	model, _ = result.Update(keyRunes(query))
	result = mustResponsePane(t, model)

	model, _ = result.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	return mustResponsePane(t, model)
}

// TestResponsePane_Search_FindsMatchInRawBody is a regression test for a
// real feature request: "/" should search the response body, jumping to
// the first match.
func TestResponsePane_Search_FindsMatchInRawBody(t *testing.T) {
	pane := setupPaneWithLines(t, 200)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line 150")

	if result.searching {
		t.Error("expected enter to close the search prompt")
	}
	if len(result.searchMatches) != 1 {
		t.Fatalf("expected exactly 1 match for \"line 150\", got %d: %v", len(result.searchMatches), result.searchMatches)
	}
	if !strings.Contains(result.viewport.View(), "line 150") {
		t.Errorf("expected the match to be scrolled into view, got:\n%s", result.viewport.View())
	}
}

// TestResponsePane_Search_FindsMatchInPrettyBody verifies search also
// works against the syntax-highlighted (ANSI-styled) pretty view, not
// just raw - a naive substring search over the ANSI-wrapped text would
// otherwise miss matches whose characters fall under different color
// spans (see searchMatchLines's use of ansi.Strip).
func TestResponsePane_Search_FindsMatchInPrettyBody(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(100)
	pane.SetHeight(20)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       `{"albumId": 1, "title": "a distinctive marker phrase"}`,
	}, &http.Request{})

	if pane.rawBody {
		t.Fatal("test setup error: expected the pretty (non-raw) view by default")
	}

	result := typeAndConfirmSearch(t, &pane, "distinctive marker phrase")

	if len(result.searchMatches) != 1 {
		t.Fatalf("expected the pretty/highlighted view to still be searchable, got %d matches", len(result.searchMatches))
	}
}

// TestResponsePane_Search_NoMatches_ReportsStatus verifies a query with no
// matches leaves the pane in a coherent state (no crash, an explanatory
// status) instead of matches from some previous, unrelated search.
func TestResponsePane_Search_NoMatches_ReportsStatus(t *testing.T) {
	pane := setupPaneWithLines(t, 20)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "nonexistent-xyz")

	if len(result.searchMatches) != 0 {
		t.Errorf("expected no matches, got %v", result.searchMatches)
	}
	if result.searchStatusMsg == "" {
		t.Error("expected a \"no matches\" status message")
	}
}

// TestResponsePane_Search_CountsEachOccurrenceOnALineSeparately is a
// regression test for a real bug: a line where the query appears more
// than once used to be counted (and jumped between via "n"/"b") as a
// single match, the same as a line with just one occurrence - each
// occurrence must be its own, independently countable/navigable match.
func TestResponsePane_Search_CountsEachOccurrenceOnALineSeparately(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(100)
	pane.SetHeight(20)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"text/plain"}},
		Body:       "needle one needle two needle three\nno match here",
	}, &http.Request{})
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, &pane, "needle")

	if len(result.searchMatches) != 3 {
		t.Fatalf("expected 3 separate matches (one per occurrence on the line), got %d: %+v", len(result.searchMatches), result.searchMatches)
	}
	for _, m := range result.searchMatches {
		if m.line != 0 {
			t.Errorf("expected every occurrence on line 0, got %+v", m)
		}
	}
	if result.searchMatches[0].start >= result.searchMatches[1].start || result.searchMatches[1].start >= result.searchMatches[2].start {
		t.Errorf("expected occurrences in left-to-right order, got %+v", result.searchMatches)
	}
}

// TestResponsePane_SearchNextPrev_CycleThroughMatches is a regression test
// for the vim/less-style "n"/"b" match navigation this was built for.
func TestResponsePane_SearchNextPrev_CycleThroughMatches(t *testing.T) {
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	// Every line contains "line ", so all 30 lines match - a simple,
	// predictable sequence to cycle through.
	result := typeAndConfirmSearch(t, pane, "line ")
	if len(result.searchMatches) != 30 {
		t.Fatalf("expected 30 matches, got %d", len(result.searchMatches))
	}
	startIdx := result.searchMatchIdx

	model, _ := result.Update(keyRunes("n"))
	afterNext := mustResponsePane(t, model)
	wantNext := (startIdx + 1) % len(afterNext.searchMatches)
	if afterNext.searchMatchIdx != wantNext {
		t.Errorf("expected \"n\" to advance to match %d, got %d", wantNext, afterNext.searchMatchIdx)
	}

	model, _ = afterNext.Update(keyRunes("b"))
	afterPrev := mustResponsePane(t, model)
	if afterPrev.searchMatchIdx != startIdx {
		t.Errorf("expected \"b\" to move back to match %d, got %d", startIdx, afterPrev.searchMatchIdx)
	}

	// "b" from the very first match must wrap around to the last one.
	model, _ = afterPrev.Update(keyRunes("b"))
	wrapped := mustResponsePane(t, model)
	if wrapped.searchMatchIdx != len(wrapped.searchMatches)-1 {
		t.Errorf("expected \"b\" to wrap to the last match (%d), got %d", len(wrapped.searchMatches)-1, wrapped.searchMatchIdx)
	}
}

// setupAdvancedSearchAtTop builds a pane with a search ("line ", matching
// every one of 200 lines - so match index N always sits at line index N,
// making expected positions trivial to compute) already advanced 50
// matches deep via "n", then jumped back to the top of the view via "gg" -
// exactly the scenario this feature was requested for: the remembered
// match-sequence position (searchMatchIdx) and the view's actual current
// position have diverged. Returns a *fresh* pane each call - ResponsePane.
// Update has a pointer receiver and mutates in place, so a single instance
// can't safely be reused across two independent "what does the next key
// press do from here" checks.
func setupAdvancedSearchAtTop(t *testing.T) (atTop *ResponsePane, advancedIdx int) {
	t.Helper()
	pane := setupPaneWithLines(t, 200)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line ")
	if len(result.searchMatches) != 200 {
		t.Fatalf("test setup error: expected 200 matches, got %d", len(result.searchMatches))
	}

	model := result
	for i := 0; i < 50; i++ {
		model, _ = model.Update(keyRunes("n"))
	}
	advancedIdx = model.searchMatchIdx
	if advancedIdx == 0 {
		t.Fatal("test setup error: expected \"n\" to have advanced the match index")
	}

	model, _ = model.Update(keyRunes("g"))
	model, _ = model.Update(keyRunes("g"))
	atTop = model
	if atTop.viewport.YOffset() != 0 {
		t.Fatalf("test setup error: expected \"gg\" to scroll to the top, got offset %d", atTop.viewport.YOffset())
	}
	return atTop, advancedIdx
}

// TestResponsePane_SearchNext_ContinuesFromRememberedPosition verifies
// "n" is unaffected by this feature: after the view moves away on its own
// (here, "gg"), "n" still continues from wherever the match sequence
// itself last left off, not from the view's new position.
func TestResponsePane_SearchNext_ContinuesFromRememberedPosition(t *testing.T) {
	atTop, advancedIdx := setupAdvancedSearchAtTop(t)

	model, _ := atTop.Update(keyRunes("n"))
	afterN := mustResponsePane(t, model)
	if afterN.searchMatchIdx != advancedIdx+1 {
		t.Errorf("expected \"n\" to continue from the remembered position (%d), got %d", advancedIdx+1, afterN.searchMatchIdx)
	}
}

// TestResponsePane_SearchNextFromPosition_ReanchorsToCurrentView is a
// regression test for a real feature request: "N" must instead re-anchor
// to wherever the view is currently scrolled to (here, the top, via
// "gg") and search from there - finding the very first match - rather
// than continuing from the remembered match-sequence position.
func TestResponsePane_SearchNextFromPosition_ReanchorsToCurrentView(t *testing.T) {
	atTop, _ := setupAdvancedSearchAtTop(t)

	model, _ := atTop.Update(keyRunes("N"))
	afterCapitalN := mustResponsePane(t, model)
	if afterCapitalN.searchMatchIdx != 0 {
		t.Errorf("expected \"N\" to re-anchor to the top of the view (match 0), got %d", afterCapitalN.searchMatchIdx)
	}
}

// TestResponsePane_SearchNextFromPosition_ContinuesWhenMatchStillVisible
// is a regression test for a real bug: "N" used to always re-anchor to
// the viewport's current top, even when the remembered match was still
// right there on screen - so a plain "/" search immediately followed by
// "N" (nothing else pressed in between) skipped straight back to the
// *same* match instead of advancing to the next one, since the current
// match's own line satisfied "the first match at or after the top of the
// view". "N" must behave exactly like "n" whenever the remembered match
// hasn't actually scrolled out of view.
func TestResponsePane_SearchNextFromPosition_ContinuesWhenMatchStillVisible(t *testing.T) {
	pane := setupPaneWithLines(t, 200)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line ")
	matchLine := result.searchMatches[result.searchMatchIdx].line
	if !result.isLineVisible(matchLine) {
		t.Fatal("test setup error: expected the just-found match to still be visible")
	}
	// Captured before Update runs: ResponsePane.Update has a pointer
	// receiver and mutates result in place, so result.searchMatchIdx
	// itself would already reflect the *post*-"N" value by the time of
	// the comparison below otherwise.
	beforeIdx := result.searchMatchIdx

	model, _ := result.Update(keyRunes("N"))
	afterN := mustResponsePane(t, model)
	if afterN.searchMatchIdx != beforeIdx+1 {
		t.Errorf("expected \"N\" to just advance to the next match (%d) since the current one is still visible, got %d", beforeIdx+1, afterN.searchMatchIdx)
	}
}

// TestResponsePane_SearchNextFromPosition_UnaffectedByOrdinaryScroll is a
// regression test for a real bug: an ordinary small scroll (j/k,
// pgup/pgdn) that never actually scrolls the remembered match out of
// view was still treated the same as a real jump elsewhere (gg/G/g{n}g) -
// re-anchoring "N"/"P" to wherever the view happened to be nudged to,
// instead of recognizing the match was still on screen and just
// continuing from it like "n"/"b" would.
func TestResponsePane_SearchNextFromPosition_UnaffectedByOrdinaryScroll(t *testing.T) {
	pane := setupPaneWithLines(t, 200)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line 100")
	if len(result.searchMatches) != 1 {
		t.Fatalf("test setup error: expected exactly one match, got %d", len(result.searchMatches))
	}

	model, _ := result.Update(keyRunes("j"))
	model, _ = model.Update(keyRunes("j"))
	scrolled := mustResponsePane(t, model)
	matchLine := scrolled.searchMatches[scrolled.searchMatchIdx].line
	if !scrolled.isLineVisible(matchLine) {
		t.Fatal("test setup error: expected the match to still be visible after a small scroll")
	}

	model, _ = scrolled.Update(keyRunes("N"))
	afterN := mustResponsePane(t, model)
	if afterN.searchMatchIdx != 0 {
		t.Errorf("expected \"N\" to still land on the only match (0) after an ordinary scroll, got %d", afterN.searchMatchIdx)
	}
}

// TestResponsePane_SearchPrevFromPosition_ReanchorsToCurrentView mirrors
// the "N" test above for "P": searching backward, re-anchored to
// wherever the view is currently scrolled to (set directly here, since
// "line " matches every line 1:1 with its own index, making the expected
// match trivial to compute) rather than the remembered match-sequence
// position.
func TestResponsePane_SearchPrevFromPosition_ReanchorsToCurrentView(t *testing.T) {
	pane := setupPaneWithLines(t, 200)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line ")
	if len(result.searchMatches) != 200 {
		t.Fatalf("test setup error: expected 200 matches, got %d", len(result.searchMatches))
	}
	result.viewport.SetYOffset(150)

	model, _ := result.Update(keyRunes("P"))
	afterP := mustResponsePane(t, model)
	if afterP.searchMatchIdx != 150 {
		t.Errorf("expected \"P\" to re-anchor to the current view position (match 150), got %d", afterP.searchMatchIdx)
	}
}

// TestResponsePane_Search_ResetsWhenContentChanges verifies switching
// tabs invalidates a previous search - its matches are line indices into
// the content that was displayed when it ran, so they'd otherwise
// silently point at the wrong lines once the content underneath changes.
func TestResponsePane_Search_ResetsWhenContentChanges(t *testing.T) {
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line 5")
	if len(result.searchMatches) == 0 {
		t.Fatal("test setup error: expected at least one match")
	}

	model, _ := result.Update(keyRunes("2")) // Headers tab
	afterTabSwitch := mustResponsePane(t, model)
	if len(afterTabSwitch.searchMatches) != 0 || afterTabSwitch.searchQuery != "" {
		t.Errorf("expected switching tabs to reset the search, got query=%q matches=%v", afterTabSwitch.searchQuery, afterTabSwitch.searchMatches)
	}
}

// TestResponsePane_Search_ResetsWhenTogglingRawChangesJSONContent is the
// raw/pretty-toggle counterpart of the tab-switch test above, using JSON
// specifically: unlike plaintext/html/xml (which formatContentByType
// never reformats to begin with, so raw and pretty highlighting the same
// bytes now come out identical - see highlightRawBody), JSON's pretty
// view re-indents it (see formatJSON), so its raw and pretty text still
// genuinely differ and toggling between them must still reset a search.
func TestResponsePane_Search_ResetsWhenTogglingRawChangesJSONContent(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(100)
	pane.SetHeight(20)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       `{"a":1,"needle":"x","c":3}`,
	}, &http.Request{})

	result := typeAndConfirmSearch(t, &pane, "needle")
	if len(result.searchMatches) == 0 {
		t.Fatal("test setup error: expected at least one match")
	}

	model, _ := result.Update(keyRunes("r")) // toggle to raw
	afterToggle := mustResponsePane(t, model)
	if len(afterToggle.searchMatches) != 0 || afterToggle.searchQuery != "" {
		t.Errorf("expected toggling raw/pretty on JSON to reset the search, got query=%q matches=%v", afterToggle.searchQuery, afterToggle.searchMatches)
	}
}

// TestResponsePane_Search_PersistsAcrossIdenticalContent verifies
// re-rendering the *same* content (e.g. switching to another tab and
// back) does NOT throw away an active search, only an actual content
// change does (see TestResponsePane_Search_ResetsWhenContentChanges).
func TestResponsePane_Search_PersistsAcrossIdenticalContent(t *testing.T) {
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line 5")
	if len(result.searchMatches) == 0 {
		t.Fatal("test setup error: expected at least one match")
	}

	// Re-populating the viewport with the exact same content it already
	// has (as switching tabs and back would, once returned to this tab)
	// must not disturb the search.
	result.updateViewportForActiveTab()

	if len(result.searchMatches) == 0 || result.searchQuery != "line 5" {
		t.Errorf("expected the search to survive a re-render of identical content, got query=%q matches=%v", result.searchQuery, result.searchMatches)
	}
}

// TestResponsePane_Search_JumpsLiveBeforeEnter is a regression test for a
// real feature request: matches must be found and jumped to as the query
// is typed (vim/less incremental search), not only once enter confirms.
func TestResponsePane_Search_JumpsLiveBeforeEnter(t *testing.T) {
	pane := setupPaneWithLines(t, 200)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	model, _ := pane.Update(keyRunes("/"))
	result := mustResponsePane(t, model)

	model, _ = result.Update(keyRunes("line 150"))
	result = mustResponsePane(t, model)

	if !result.searching {
		t.Fatal("test setup error: expected the prompt to still be open")
	}
	if len(result.searchMatches) != 1 {
		t.Fatalf("expected the match to already be found before enter, got %d matches", len(result.searchMatches))
	}
	if !strings.Contains(result.viewport.View(), "line 150") {
		t.Errorf("expected the live search to already have scrolled to the match, got:\n%s", result.viewport.View())
	}
}

// TestResponsePane_Search_ForwardFromPositionWrapsToStart is a regression
// test for the exact search semantics requested: search forward from the
// current position first, and only if nothing matches below, wrap around
// and jump to the first match from the top.
func TestResponsePane_Search_ForwardFromPositionWrapsToStart(t *testing.T) {
	pane := setupPaneWithLines(t, 200)
	pane.rawBody = true
	pane.updateViewportForActiveTab()
	pane.viewport.SetYOffset(100) // sit in the middle of the content

	// "line 3" matches "line 3" and "line 3X" for X in 0-9 - line indices
	// 2 and 29-38 - all well above the starting position (100), so a
	// forward-only search finds nothing there and must wrap to the top.
	result := typeAndConfirmSearch(t, pane, "line 3")
	if len(result.searchMatches) == 0 {
		t.Fatal("test setup error: expected at least one match for \"line 3\"")
	}
	matchedLine := result.searchMatches[result.searchMatchIdx].line
	if matchedLine >= 100 {
		t.Errorf("expected the search to wrap to a match above the starting position (100), landed on line index %d", matchedLine)
	}
}

// TestResponsePane_Search_HighlightsMatches verifies matched lines are
// visually highlighted in the rendered output, and that the current match
// is styled distinctly from the rest.
func TestResponsePane_Search_HighlightsMatches(t *testing.T) {
	// lipgloss detects whether its output is an actual terminal and
	// silently renders plain, unstyled text otherwise (as it would under
	// `go test`, with no TTY attached) - force a color profile for this
	// check so the highlight styling (see highlightMatches/styles.go)
	// actually gets a chance to render, restoring it after.
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line ")
	if len(result.searchMatches) < 2 {
		t.Fatal("test setup error: expected multiple matches")
	}

	out := result.viewport.View()
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("expected the rendered view to contain highlight styling (ANSI escapes) for matches, got plain text:\n%s", out)
	}
}

// TestResponsePane_Search_HighlightsOnlyThePattern is a regression test
// for a real bug: highlighting used to cover an entire matching line,
// not just the text that was actually searched for - the rest of the
// line (everything besides the query itself) must stay as plain,
// unstyled text.
func TestResponsePane_Search_HighlightsOnlyThePattern(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(150)
	pane.SetHeight(20)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"text/plain"}},
		Body:       "prefixmarker NEEDLE suffixmarker",
	}, &http.Request{})
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, &pane, "needle")

	out := result.viewport.View()
	if !strings.Contains(out, "\x1b[") {
		t.Fatal("test setup error: expected highlight styling to render")
	}
	// The surrounding words must appear verbatim, as plain text - if
	// highlighting covered the whole line, these exact substrings
	// wouldn't appear unbroken; they'd be interleaved with (or wrapped
	// entirely inside) the highlight style's own escape codes instead.
	if !strings.Contains(out, "prefixmarker") {
		t.Errorf("expected \"prefixmarker\" to remain as plain text outside the highlight, got:\n%s", out)
	}
	if !strings.Contains(out, "suffixmarker") {
		t.Errorf("expected \"suffixmarker\" to remain as plain text outside the highlight, got:\n%s", out)
	}
	if strings.Contains(out, currentMatchStyle.Render("prefixmarker")) {
		t.Errorf("expected only \"needle\" to be highlighted, not the surrounding text")
	}
}

// TestResponsePane_Search_SupportsRegexMetacharacters is a regression test
// for a real feature request: "/" should accept regular expressions, not
// just literal substrings - "." "?" "*" and friends (full Go RE2 syntax,
// via compileSearchQuery) should behave as regex metacharacters, not be
// matched literally.
func TestResponsePane_Search_SupportsRegexMetacharacters(t *testing.T) {
	pane := setupPaneWithLines(t, 20)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	// "line 1.$" anchors to lines ending in "1" followed by exactly one
	// more digit - matching "line 10".."line 19", not "line 1" itself
	// (no trailing digit) or "line 100"-style lines (out of range here).
	result := typeAndConfirmSearch(t, pane, "line 1.$")

	if len(result.searchMatches) != 10 {
		t.Fatalf("expected 10 matches (line 10..line 19), got %d: %v", len(result.searchMatches), result.searchMatches)
	}
	if !strings.Contains(result.viewport.View(), "line 10") {
		t.Errorf("expected the first match to be scrolled into view, got:\n%s", result.viewport.View())
	}
}

// TestResponsePane_Search_InvalidRegexFallsBackToLiteral is a regression
// test for graceful degradation: a query that doesn't parse as valid regex
// (unbalanced parens are common in real body/header text, e.g. searching
// for a literal "(default)") must still work, matched as a plain literal
// string instead of the search silently producing zero matches or the
// pane crashing on a regexp.MustCompile panic.
func TestResponsePane_Search_InvalidRegexFallsBackToLiteral(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	pane.SetWidth(100)
	pane.SetHeight(20)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"text/plain"}},
		Body:       "retries (default)",
	}, &http.Request{})
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, &pane, "(default")

	if len(result.searchMatches) != 1 {
		t.Fatalf("expected the unbalanced-paren query to still match literally, got %d matches: %v", len(result.searchMatches), result.searchMatches)
	}
}

// spansFor is a small test helper mirroring what findSearchMatches would
// produce for a single line, without needing a full ResponsePane/content
// string just to compute byte-offset spans for highlightLineSpans.
func spansFor(t *testing.T, plain, query string, style lipgloss.Style) []styledSpan {
	t.Helper()
	re := compileSearchQuery(query)
	var spans []styledSpan
	for _, span := range re.FindAllStringIndex(plain, -1) {
		spans = append(spans, styledSpan{span[0], span[1], style})
	}
	return spans
}

// TestHighlightLineSpans_HighlightsEveryRegexSpan verifies a pattern that
// matches more than once per line (not just a fixed literal repeated) gets
// every span highlighted, each independently, using its own found bounds
// rather than a fixed literal length.
func TestHighlightLineSpans_HighlightsEveryRegexSpan(t *testing.T) {
	line := "id=1 count=42 total=100"
	out := highlightLineSpans(line, spansFor(t, line, `\d+`, otherMatchStyle))

	for _, want := range []string{"1", "42", "100"} {
		if !strings.Contains(out, otherMatchStyle.Render(want)) {
			t.Errorf("expected %q to be highlighted as its own span, got:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "id=") || !strings.Contains(out, "count=") || !strings.Contains(out, "total=") {
		t.Errorf("expected the non-matching text to remain plain, got:\n%s", out)
	}
}

// TestHighlightLineSpans_PreservesSurroundingColor is a regression test
// for a real bug: highlighting a match used to re-render its *entire*
// line from plain, ansi.Strip'd text, discarding any syntax coloring
// (e.g. chroma's pretty-body highlighting) on the rest of that line, not
// just the matched span - a matched line would come out entirely flat/
// white instead of keeping its existing colors around the highlight.
func TestHighlightLineSpans_PreservesSurroundingColor(t *testing.T) {
	colored := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render("prefix") + "NEEDLE" +
		lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render("suffix")
	plain := ansi.Strip(colored)

	out := highlightLineSpans(colored, spansFor(t, plain, "needle", otherMatchStyle))

	if ansi.Strip(out) != "prefixNEEDLEsuffix" {
		t.Fatalf("expected the line's text to survive intact, got %q", ansi.Strip(out))
	}
	if !strings.Contains(out, "\x1b[38;5;255;48;5;58m") {
		t.Errorf("expected the match itself to carry the highlight style, got:\n%s", out)
	}
	if !strings.Contains(out, "\x1b[38;5;212m") {
		t.Errorf("expected the surrounding text's original color to survive, got:\n%s", out)
	}
}

// TestHighlightMatches_StaysFastWithManyHitsOnOneLine is a regression
// test for a real performance bug: highlightLineSpans used to call
// ansi.Cut once per gap between matches, and ansi.Cut re-scans its input
// from the start on every call - with many matches on one long line, that
// made highlighting (re-run on every search keystroke - see liveSearch)
// quadratic in the number of matches, turning a search with lots of hits
// into a multi-second stall. 1000 hits across a ~30KB pretty-printed body
// must still highlight well under what a keystroke's frame budget allows.
func TestHighlightMatches_StaysFastWithManyHitsOnOneLine(t *testing.T) {
	var body strings.Builder
	body.WriteString("{")
	for i := range 200 {
		fmt.Fprintf(&body, `"key%d": "value with needle needle needle needle needle text", `, i)
	}
	body.WriteString("}")
	pretty := formatContentByType(body.String(), "application/json", "onedark")

	matches := findSearchMatches(pretty, "needle")
	if len(matches) != 1000 {
		t.Fatalf("test setup error: expected 1000 matches, got %d", len(matches))
	}

	start := time.Now()
	highlightMatches(pretty, "needle", matches, 0)
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Errorf("highlighting 1000 matches took %s, want well under 200ms", elapsed)
	}
}

// TestResponsePane_View_ShowsContentWhileSearching is a regression test
// for a real feature request: opening "/" must NOT hide the response
// content behind a full-screen prompt (as the save-to-file prompt does) -
// it should render inline, in place of the footer hint, with the content
// still visible and scrollable behind it.
func TestResponsePane_View_ShowsContentWhileSearching(t *testing.T) {
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	model, _ := pane.Update(keyRunes("/"))
	result := mustResponsePane(t, model)
	model, _ = result.Update(keyRunes("line 5"))
	result = mustResponsePane(t, model)

	out := result.View()
	if !strings.Contains(out, "line 5") {
		t.Errorf("expected the response content to remain visible while searching, got:\n%s", out)
	}
	if strings.Contains(out, "Save raw response body") {
		t.Errorf("did not expect the save-to-file prompt's framing, got:\n%s", out)
	}
}

// TestResponsePane_SearchEsc_ClosesPromptWithoutDiscardingPreviousSearch
// verifies esc cancels only the *in-progress edit* of a new query, the
// same way it cancels the save-to-file prompt without discarding the
// response itself.
func TestResponsePane_SearchEsc_ClosesPromptWithoutDiscardingPreviousSearch(t *testing.T) {
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line 5")

	model, _ := result.Update(keyRunes("/"))
	reopened := mustResponsePane(t, model)
	model, _ = reopened.Update(keyRunes("something else"))
	typed := mustResponsePane(t, model)

	model, _ = typed.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	afterEsc := mustResponsePane(t, model)

	if afterEsc.searching {
		t.Error("expected esc to close the search prompt")
	}
	if afterEsc.searchQuery != "line 5" {
		t.Errorf("expected esc to leave the previous search intact, got query=%q", afterEsc.searchQuery)
	}
}
