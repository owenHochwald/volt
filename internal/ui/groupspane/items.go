package groupspane

import (
	"fmt"

	"github.com/owenHochwald/Volt/internal/http"
)

type groupItem struct {
	group  http.Group
	server string // display label: assigned server's name, or "(none)"
}

func (i groupItem) Title() string       { return i.group.Name }
func (i groupItem) Description() string { return "server: " + i.server }
func (i groupItem) FilterValue() string { return i.group.Name }

type serverItem struct {
	server http.Server
}

func (i serverItem) Title() string { return i.server.Name }
func (i serverItem) Description() string {
	if len(i.server.Headers) == 0 {
		return i.server.BaseURL
	}
	return fmt.Sprintf("%s · %d header(s)", i.server.BaseURL, len(i.server.Headers))
}
func (i serverItem) FilterValue() string { return i.server.Name }

func (m GroupsPane) selectedGroup() (groupItem, bool) {
	if item := m.groups.SelectedItem(); item != nil {
		if gi, ok := item.(groupItem); ok {
			return gi, true
		}
	}
	return groupItem{}, false
}

func (m GroupsPane) selectedServer() (serverItem, bool) {
	if item := m.servers.SelectedItem(); item != nil {
		if si, ok := item.(serverItem); ok {
			return si, true
		}
	}
	return serverItem{}, false
}
