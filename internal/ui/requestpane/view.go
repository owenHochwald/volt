package requestpane

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/ui"
)

// footerHelpStyle renders this pane's pinned help hint without
// ui.HelpStyle's Margin(1, 2) - that margin bakes in a blank line above
// *and* below the text (2 extra rows just for a hint line), which in this
// pane's already-tight footer budget left barely any height for
// FieldsViewport to show actual fields in.
var footerHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// errorTextStyle renders a blocked-send reason (see ValidateParams) in the
// footer, distinct from the plain help hint below it.
var errorTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

// fieldOffset pairs a focusable with the 0-based line it starts on within
// fieldsContent()'s rendered output - used to auto-scroll FieldsViewport so
// whatever's currently focused (e.g. a field of a just-added array item,
// several screens down) stays visible instead of receiving keystrokes
// off-screen. See RequestPane.scrollToFocused.
type fieldOffset struct {
	focusable ui.Focusable
	line      int
}

// fieldsContent renders everything FieldsViewport scrolls: method+URL,
// name, path params, body params, headers, and (in load-test mode) the
// load-test config fields. Pulled out of View() so it can also be called
// from Update() - PageUp/PageDown there must refresh the viewport's
// content before scrolling it, since View() has a value receiver and so
// can only ever mutate its own local, discarded copy of FieldsViewport;
// left solely to View(), the persisted FieldsViewport a later Update()
// scrolls would still have the empty content it was constructed with, so
// maxYOffset() would always read 0 and pgdown would silently do nothing.
// The returned offsets locate every focusable field within that content
// (see scrollToFocused).
func (m RequestPane) fieldsContent() (string, []fieldOffset) {
	methodRendered := m.MethodSelector.GetStyle().Render(m.MethodSelector.Current())
	primaryLine := lipgloss.JoinHorizontal(lipgloss.Left, methodRendered, " ", m.URLInput.View())

	pathParamsBlock, pathOffsets := renderParamSection("Path Params", m.PathParams, m.ShowRequiredErrors)

	nameLabel := fieldGroupLabelStyle.Render("Name ")
	nameLine := lipgloss.JoinHorizontal(lipgloss.Left, nameLabel, m.NameInput.View())

	headersLabel := fieldGroupLabelStyle.Render("Headers ")
	headersLine := lipgloss.JoinHorizontal(lipgloss.Left, headersLabel, m.Headers.View())

	// An imported request with a known JSON body schema gets structured,
	// individually-fillable fields (like path params) instead of the
	// freeform body textarea. An "array of objects" field (see ArrayParam)
	// is rendered separately below, since it's a repeatable group of
	// sub-fields rather than a single value - bodyBlock stays empty when
	// there are no scalar BodyParams to show alongside it.
	var bodyBlock string
	var bodyOffsets []fieldOffset
	if len(m.BodyParams) > 0 {
		bodyBlock, bodyOffsets = renderParamSection("Body Params", m.BodyParams, m.ShowRequiredErrors)
	} else if len(m.ArrayParams) == 0 {
		bodyLabel := fieldGroupLabelStyle.Render("Body    ")
		bodyBlock = lipgloss.JoinHorizontal(lipgloss.Left, bodyLabel, m.Body.View())
	}
	arrayParamsBlock, arrayOffsets := renderArrayParamSection(m.ArrayParams, m.ShowRequiredErrors)

	// Field order: server, name, method+URL, path params, body params,
	// array params, headers, then mode-specific content (load-test config
	// fields, or nothing). Server, then name, then the request itself -
	// name is rarely touched once a request is built from an imported
	// schema, so it sits above the fields that actually matter each time,
	// reachable with one shift+tab if it's ever needed (see
	// reinitRequestPane's defaultFocusIndex, which starts focus on
	// method+URL, not here, for the same reason). Each present block is
	// separated from the next by a blank line; pathParamsBlock/bodyBlock/
	// arrayParamsBlock are each skipped entirely (no double blank line)
	// when there's nothing to show.
	//
	// offsets is built alongside elements, converting each block's own
	// (relative to its own first line) offsets to absolute line numbers by
	// adding the running line count so far - lipgloss.Height() rather than
	// a hardcoded line-count guess, so this can't silently drift out of
	// sync if a block's own rendering ever changes shape.
	// A request with no group (see ShowServerSelector) has no group to
	// scope its choice of server by - and, per its own group_servers
	// membership becoming the only allowed set for a grouped request (see
	// rebuildServerSelector), there's nothing meaningful left for an
	// ungrouped request to pick anyway - so the field is left out
	// entirely rather than shown picking from every server in the
	// database, unscoped.
	var elements []string
	var offsets []fieldOffset
	if m.ShowServerSelector() {
		serverLabel := fieldGroupLabelStyle.Render("Server ")
		serverLine := lipgloss.JoinHorizontal(lipgloss.Left, serverLabel, m.ServerSelector.View())
		elements = []string{serverLine, "", nameLine, "", primaryLine}
		offsets = []fieldOffset{
			{m.ServerSelector, 0},
			{m.NameInput, 2},
			{m.MethodSelector, 4},
			{m.URLInput, 4},
		}
	} else {
		elements = []string{nameLine, "", primaryLine}
		offsets = []fieldOffset{
			{m.NameInput, 0},
			{m.MethodSelector, 2},
			{m.URLInput, 2},
		}
	}
	lineCount := func() int { return lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, elements...)) }

	if pathParamsBlock != "" {
		base := lineCount()
		elements = append(elements, "", pathParamsBlock)
		for _, o := range pathOffsets {
			offsets = append(offsets, fieldOffset{o.focusable, base + 1 + o.line})
		}
	}
	if bodyBlock != "" {
		base := lineCount()
		elements = append(elements, "", bodyBlock)
		for _, o := range bodyOffsets {
			offsets = append(offsets, fieldOffset{o.focusable, base + 1 + o.line})
		}
	}
	if arrayParamsBlock != "" {
		base := lineCount()
		elements = append(elements, "", arrayParamsBlock)
		for _, o := range arrayOffsets {
			offsets = append(offsets, fieldOffset{o.focusable, base + 1 + o.line})
		}
	}
	offsets = append(offsets, fieldOffset{m.Headers, lineCount() + 1})
	elements = append(elements, "", headersLine)

	if m.LoadTestMode {
		ltConcurrencyLabel := ui.LabelStyle.Render("Concurrency:    ")
		ltConcurrencyLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltConcurrencyLabel, m.LoadTestConcurrency.View())

		ltTotalLabel := ui.LabelStyle.Render("Total Requests: ")
		ltTotalLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltTotalLabel, m.LoadTestTotalReqs.View())

		ltQPSLabel := ui.LabelStyle.Render("QPS (limit):    ")
		ltQPSLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltQPSLabel, m.LoadTestQPS.View())

		ltTimeoutLabel := ui.LabelStyle.Render("Timeout:        ")
		ltTimeoutLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltTimeoutLabel, m.LoadTestTimeout.View())

		base := lineCount()
		offsets = append(offsets,
			fieldOffset{m.LoadTestConcurrency, base + 2},
			fieldOffset{m.LoadTestTotalReqs, base + 3},
			fieldOffset{m.LoadTestQPS, base + 4},
			fieldOffset{m.LoadTestTimeout, base + 5},
		)

		elements = append(elements,
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Render("Load Test Configuration:"),
			ltConcurrencyLine,
			ltTotalLine,
			ltQPSLine,
			ltTimeoutLine,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, elements...), offsets
}

// scrollToFocused scrolls FieldsViewport so the currently focused field is
// visible, if it isn't already - called after anything that moves focus or
// changes the field layout (NextField/PrevField, and adding/removing an
// array item via its count control). Without this, tabbing to a field
// below the viewport's current scroll position - or one that only exists
// after adding an array item - left it receiving keystrokes while
// completely invisible, the only sign of trouble being pgdn revealing it
// afterward.
func (m *RequestPane) scrollToFocused() {
	if m.FocusManager == nil {
		return
	}
	content, offsets := m.fieldsContent()
	m.FieldsViewport.SetContent(content)
	// Recomputed exactly (matching View()'s own math) rather than trusting
	// SetHeight's approximation - that approximation is fine for
	// pgup/pgdown's jump size, but ensureLineVisible below needs the real
	// height or it can scroll one line short, clipping exactly the field
	// this is trying to reveal.
	m.FieldsViewport.Height = max(m.Height-lipgloss.Height(m.renderFooter()), 1)

	current := m.FocusManager.Current()
	for _, o := range offsets {
		if o.focusable == current {
			m.ensureLineVisible(o.line)
			return
		}
	}
}

// ensureLineVisible scrolls just enough to bring line within the viewport's
// visible window, without otherwise disturbing the current scroll position.
func (m *RequestPane) ensureLineVisible(line int) {
	height := m.FieldsViewport.Height
	if height <= 0 {
		return
	}
	if line < m.FieldsViewport.YOffset {
		m.FieldsViewport.SetYOffset(line)
	} else if line >= m.FieldsViewport.YOffset+height {
		m.FieldsViewport.SetYOffset(line - height + 1)
	}
}

// renderFooter renders the pinned button/stopwatch/errors/help-hint block
// below FieldsViewport. Pulled out of View() so its height can also be
// computed exactly from Update() (see refreshViewportHeight) instead of
// only approximated there - the footer's height varies with
// RequestInProgress/LoadTestMode/ParseErrors, so a fixed guess (see the old
// approxFooterLines) drifts from what's actually rendered whenever any of
// those change, which used to only blur pgup/pgdown's jump size but now
// would also throw off scrollToFocused's exact positioning.
func (m RequestPane) renderFooter() string {
	// Render button based on state
	var button string
	var stopwatchCount string
	if m.RequestInProgress {
		if m.LoadTestMode {
			button = ui.FocusedButton.Render("Running Load Test...")
		} else {
			button = ui.FocusedButton.Render("Sending...")
			elapsed := m.Stopwatch.Elapsed()
			milliseconds := elapsed.Milliseconds()
			seconds := float64(milliseconds) / 1000.0
			stopwatchCount = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Render(fmt.Sprintf("%.3fs", seconds))
		}
	} else if m.SubmitButton.IsFocused() {
		button = ui.FocusedButton.Render("→ Send")
	} else {
		button = ui.UnfocusedButton.Render("→ Send")
	}

	// Render mode-specific help hint, capped to m.Width via MaxWidth (which
	// truncates - unlike Width, which wraps to a second line). Without this,
	// the hint's natural length (well over 100 chars) exceeded m.Width on
	// anything but a very wide terminal, and got silently *wrapped* one
	// layer up when view.go's outer box applied its own Width() - adding a
	// line the box's Height() calculation (see layout.go's requestHeight)
	// knew nothing about, so the request/response column overshot the
	// height reserved for it and fell out of alignment with the sidebar.
	footerWidth := max(m.Width, 1)
	var helpText string
	if m.LoadTestMode {
		helpText = footerHelpStyle.MaxWidth(footerWidth).Render("ctrl+l: exit load test mode • tab/↑/↓: navigate • pgup/pgdn: scroll • ctrl+p: start load test")
	} else {
		helpText = footerHelpStyle.MaxWidth(footerWidth).Render("ctrl+l: load test mode • tab/↑/↓: navigate • pgup/pgdn: scroll • ←/→ or h/l: change method • ctrl+p: send • ctrl+x: reset • enter/→: accept URL • ctrl+s: save")
	}

	// A blocked send (a body param outside its schema's min/maxLength or
	// minimum/maximum - see ValidateParams) leaves its reasons in
	// ParseErrors; show them so "nothing happened" has a visible cause -
	// each capped to footerWidth for the same reason as helpText above.
	var errorsBlock string
	if len(m.ParseErrors) > 0 {
		lines := make([]string, len(m.ParseErrors))
		for i, e := range m.ParseErrors {
			lines[i] = errorTextStyle.MaxWidth(footerWidth).Render("! " + e)
		}
		errorsBlock = "\n" + lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	// stopwatchCount/errorsBlock are joined only when they actually have
	// something to show - each is otherwise "" (not sending, no parse
	// errors, the common case), and including an empty string as its own
	// JoinVertical element still renders it as a blank line, same as any
	// other element - so unconditionally including both used to leave two
	// blank lines between the button and helpText whenever neither had
	// anything to say.
	elements := []string{"", button}
	if stopwatchCount != "" {
		elements = append(elements, stopwatchCount)
	}
	if errorsBlock != "" {
		elements = append(elements, errorsBlock)
	}
	elements = append(elements, helpText)
	return lipgloss.JoinVertical(lipgloss.Left, elements...)
}

// View renders the request pane
func (m RequestPane) View() string {
	// Size the viewport from the footer's *actual* rendered height rather
	// than a guessed constant, so it never over- or under-reserves space -
	// and clip defensively with MaxHeight in case content is ever taller
	// than m.Height regardless (e.g. m.Height not set yet). FieldsViewport
	// can be taller than the pane's box with enough fields (a wide imported
	// body schema, many path params) - unlike a plain lipgloss.Height()
	// (which only pads, never truncates, and would corrupt the rest of
	// the app's layout once content overflowed it), the viewport always
	// clips to exactly its Height and can be scrolled (pgup/pgdown) to
	// reach whatever doesn't fit.
	footer := m.renderFooter()
	m.FieldsViewport.Height = max(m.Height-lipgloss.Height(footer), 1)
	content, _ := m.fieldsContent()
	m.FieldsViewport.SetContent(content)

	return lipgloss.NewStyle().MaxHeight(max(m.Height, 1)).Render(
		lipgloss.JoinVertical(lipgloss.Left, m.FieldsViewport.View(), footer),
	)
}

// fieldGroupLabelStyle marks every top-level field/section name in the
// pane (Server, Name, Path Params, Body/Body Params, Array Params,
// Headers - not Method+URL, which already gets its own per-method
// coloring, and not an individual param's own label inside one of these
// sections, which stays in the dimmer ui.LabelStyle) - so they read as a
// distinct, scannable column of names down the pane instead of blending
// into the same dim gray as everything else.
var fieldGroupLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("140")).Bold(true)

// requiredFieldStyle marks a required field's label - shown this way at all
// times as a hint regardless of whether it's currently empty (unlike
// errorTextStyle, which only appears once something's actually wrong).
var requiredFieldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)

// paramLabel renders a param's "name (type)" label, in requiredFieldStyle
// when the field is required so it's always visually distinct from an
// optional one - not just once it's flagged empty at send time (see
// RequiredFieldErrors).
func paramLabel(name, fieldType string, required bool) string {
	text := name
	if fieldType != "" {
		text += " (" + fieldType + ")"
	}
	if required {
		return requiredFieldStyle.Render(text + " ")
	}
	return ui.LabelStyle.Render(text + " ")
}

// renderParamSection renders a titled block with one "name [input]" line
// per param - used for both Path Params (from the URL) and Body Params
// (from an imported JSON schema). The title makes the two sections, and
// the plain fields around them, visually distinct from each other. The
// returned offsets are relative to this block's own first line (0), for
// the caller to translate into absolute lines within the full fieldsContent
// (see fieldsContent). showRequired is RequestPane.ShowRequiredErrors,
// threaded through to fieldError - see its doc comment.
func renderParamSection(title string, params []Param, showRequired bool) (string, []fieldOffset) {
	if len(params) == 0 {
		return "", nil
	}
	lines := make([]string, 0, len(params)+1)
	lines = append(lines, fieldGroupLabelStyle.Render(title+":"))
	offsets := make([]fieldOffset, len(params))
	for i, p := range params {
		label := "  " + paramLabel(p.Name, p.DisplayType(), p.Required)

		var field, comment string
		var focusable ui.Focusable
		if p.Enum != nil {
			// A fixed-choice picker (including a boolean's true/false
			// switch) can't produce an invalid value, so it never needs
			// the red-highlight/comment treatment below.
			field = p.Enum.View()
			focusable = p.Enum
		} else {
			// p is a loop copy (ranging by value), so setting TextStyle
			// here only affects this render pass, same as before this
			// function also started returning offsets - but the pointer
			// recorded for scrollToFocused must be the *real* one
			// paramFocusables hands the focus cycle (&params[i].Input),
			// not &p.Input (a copy's address, matching nothing).
			if msg, ok := fieldError(p, showRequired); !ok {
				p.Input.TextStyle = errorTextStyle
				comment = errorTextStyle.Render("  ! " + msg)
			}
			field = p.Input.View()
			focusable = &params[i].Input
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, label, field, comment))
		offsets[i] = fieldOffset{focusable, i + 1}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...), offsets
}

// renderArrayParamSection renders one titled block per array-typed body
// param (see ArrayParam): its count control (‹ N item(s) ›, cycled with
// left/right), then each current item indented under the block - a
// "complex" array (its schema's "items" was a $ref to an object) gets an
// "Item N:" sub-heading with one "name (type) [input]" line per named
// sub-field below it; a "simple" array (see ArrayParam.IsSimple) skips
// both - there's exactly one unnamed field per item, so "Item N: [input]"
// on a single line says everything the "Item N:" heading and a "(type)"
// label would otherwise say separately. Both cases get the same inline
// red-highlight/comment treatment invalid fields get in renderParamSection.
// The returned offsets are relative to this block's own first line (0).
// showRequired is RequestPane.ShowRequiredErrors, threaded through to
// fieldError - see its doc comment.
func renderArrayParamSection(arrays []ArrayParam, showRequired bool) (string, []fieldOffset) {
	if len(arrays) == 0 {
		return "", nil
	}
	blocks := make([]string, len(arrays))
	var offsets []fieldOffset
	blockStart := 0
	for ai := range arrays {
		a := &arrays[ai]
		lines := make([]string, 0, 2+len(a.Items)*3)
		lines = append(lines, fieldGroupLabelStyle.Render(fmt.Sprintf("%s (array of %s):", a.Name, a.itemLabel())))
		lines = append(lines, "  "+a.CountControl.View())
		offsets = append(offsets, fieldOffset{a.CountControl, blockStart + 1})

		line := 2
		for ii, item := range a.Items {
			if a.IsSimple() {
				f := item.Fields[0]
				label := ui.LabelStyle.Render(fmt.Sprintf("  Item %d: ", ii+1))

				var field, comment string
				var focusable ui.Focusable
				if f.Enum != nil {
					field = f.Enum.View()
					focusable = f.Enum
				} else {
					if msg, ok := fieldError(f, showRequired); !ok {
						f.Input.TextStyle = errorTextStyle
						comment = errorTextStyle.Render("  ! " + msg)
					}
					field = f.Input.View()
					focusable = &a.Items[ii].Fields[0].Input
				}
				lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, label, field, comment))
				offsets = append(offsets, fieldOffset{focusable, blockStart + line})
				line++
				continue
			}

			lines = append(lines, "", ui.LabelStyle.Render(fmt.Sprintf("  Item %d:", ii+1)))
			line += 2
			for fi, f := range item.Fields {
				label := "    " + paramLabel(f.Name, f.DisplayType(), f.Required)

				var field, comment string
				var focusable ui.Focusable
				if f.Enum != nil {
					field = f.Enum.View()
					focusable = f.Enum
				} else {
					if msg, ok := fieldError(f, showRequired); !ok {
						f.Input.TextStyle = errorTextStyle
						comment = errorTextStyle.Render("  ! " + msg)
					}
					field = f.Input.View()
					focusable = &a.Items[ii].Fields[fi].Input
				}
				lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, label, field, comment))
				offsets = append(offsets, fieldOffset{focusable, blockStart + line})
				line++
			}
		}
		blocks[ai] = lipgloss.JoinVertical(lipgloss.Left, lines...)
		blockStart += line
		if ai < len(arrays)-1 {
			blockStart++ // blank line between array blocks, added below
		}
	}

	elements := make([]string, 0, len(blocks)*2)
	for i, b := range blocks {
		if i > 0 {
			elements = append(elements, "")
		}
		elements = append(elements, b)
	}
	return lipgloss.JoinVertical(lipgloss.Left, elements...), offsets
}
