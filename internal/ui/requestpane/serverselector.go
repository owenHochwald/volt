package requestpane

import "github.com/owenHochwald/Volt/internal/http"

// noServerOption is the ServerSelector's choice for "this request doesn't
// pick a server directly" - it may still resolve one indirectly through its
// group.
const noServerOption = "(none)"

// rebuildServerSelector refreshes Servers from the database and rebuilds
// ServerSelector over it, so servers added/renamed/removed via the Groups &
// Servers modal show up next time a request is loaded, preserving
// m.Request's current ServerID (if any) as the selector's starting value.
// Called from reinitRequestPane (loading a request) and SetupRequestPane
// (initial construction).
func (m *RequestPane) rebuildServerSelector() {
	var servers []http.Server
	if m.DB != nil {
		servers, _ = m.DB.ListServers()
	}
	m.Servers = servers

	options := make([]string, 0, len(servers)+1)
	options = append(options, noServerOption)
	initial := noServerOption
	for _, s := range servers {
		options = append(options, s.Name)
		if m.Request != nil && m.Request.ServerID != nil && s.ID == *m.Request.ServerID {
			initial = s.Name
		}
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
