package requestpane

import "github.com/owenHochwald/Volt/internal/http"

// noServerOption is the ServerSelector's choice for "this request doesn't
// pick a server directly" - it may still resolve one indirectly through its
// group.
const noServerOption = "(none)"

// ShowServerSelector reports whether this request's ServerSelector field
// is meaningful at all: only a request that belongs to a group has a
// restricted, sensible set of servers to pick from (see
// rebuildServerSelector) - an ungrouped ("clean") request has no group to
// scope a choice by, and typically addresses its server directly via its
// own absolute URL anyway, so there's nothing useful for the field to
// offer. Checked by fieldsContent (whether to render/focus it at all) and
// firstErrorFocusIndex (the focus-index prefix it accounts for).
func (m *RequestPane) ShowServerSelector() bool {
	return m.Request != nil && m.Request.GroupID != nil
}

// rebuildServerSelector refreshes Servers from the database and rebuilds
// ServerSelector over it, so servers added/renamed/removed via the Groups &
// Servers modal show up next time a request is loaded, preserving
// m.Request's current ServerID (if any) as the selector's starting value.
// Called from reinitRequestPane (loading a request) and SetupRequestPane
// (initial construction).
//
// A request that belongs to a group only ever offers *that group's*
// allowed servers (see storage.ListGroupServers/AddGroupServer) - not
// every server in the database - so a request can't accidentally resolve
// against a server that belongs to a different API entirely. An ungrouped
// request has no group to scope by, and its ServerSelector isn't even
// shown (see ShowServerSelector), so there's no list worth building for
// it either.
func (m *RequestPane) rebuildServerSelector() {
	var servers []http.Server
	var groupDefaultServerID *int64
	if m.DB != nil && m.ShowServerSelector() {
		servers, _ = m.DB.ListGroupServers(*m.Request.GroupID)
		if group, err := m.DB.GetGroupByID(*m.Request.GroupID); err == nil && group != nil {
			groupDefaultServerID = group.ServerID
		}
	}
	m.Servers = servers

	// A request's own ServerID directly overrides the group's default,
	// but when it's unset (the picker showing "(none)"), resolution still
	// falls through to the group's default server rather than genuinely
	// using none at all (see ResolveEffectiveServer) - so offering
	// "(none)" as the starting choice whenever the group actually has a
	// default was misleading: the request went out to that default
	// regardless of what the picker showed. "(none)" is only offered (and
	// only ever the starting choice) when the group genuinely has no
	// default to fall back to, where it's accurate - no server really is
	// used until one's picked directly.
	includeNone := groupDefaultServerID == nil
	options := make([]string, 0, len(servers)+1)
	initial := ""
	if includeNone {
		options = append(options, noServerOption)
		initial = noServerOption
	}
	for _, s := range servers {
		options = append(options, s.Name)
		switch {
		case m.Request != nil && m.Request.ServerID != nil && s.ID == *m.Request.ServerID:
			initial = s.Name
		case initial == "" && groupDefaultServerID != nil && s.ID == *groupDefaultServerID:
			initial = s.Name
		}
	}
	// Defensive fallback: a group with a default (includeNone == false)
	// should always have that default among servers too (SetGroupServer
	// guarantees it), so this shouldn't normally trigger - but rather than
	// ever leave the picker starting unset (a different, deliberate
	// meaning elsewhere - see EnumSelector.allowUnset), fall back to
	// whatever's first.
	if initial == "" && len(servers) > 0 {
		initial = servers[0].Name
	}
	m.ServerSelector = NewEnumSelector(options, initial)
}

// serverIDByName looks up a server's ID by name among the pane's currently
// loaded Servers, for turning ServerSelector.Value() back into the
// http.Request.ServerID actually persisted.
func (m *RequestPane) serverIDByName(name string) *int64 {
	for _, s := range m.Servers {
		if s.Name == name {
			id := s.ID
			return &id
		}
	}
	return nil
}
