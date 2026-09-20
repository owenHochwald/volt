package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

type RequestsLoadingMsg struct {
	Requests []http.Request
	Err      error
}

type RequestSavedMsg struct {
	Request *http.Request
	Err     error
}

type RequestDeletedMsg struct {
	ID  int64
	Err error
}

type SetRequestPaneRequestMsg struct {
	Request *http.Request
}

func SetRequestPaneRequestCmd(request *http.Request) tea.Cmd {
	return func() tea.Msg {
		return SetRequestPaneRequestMsg{
			Request: request,
		}
	}
}

func DeleteRequestCmd(db *storage.SQLiteStorage, id int64) tea.Cmd {
	return func() tea.Msg {
		err := db.Delete(id)
		return RequestDeletedMsg{
			ID:  id,
			Err: err,
		}
	}
}

func SaveRequestCmd(db *storage.SQLiteStorage, request *http.Request) tea.Cmd {
	return func() tea.Msg {
		err := db.Save(request)
		return RequestSavedMsg{
			Request: request,
			Err:     err,
		}
	}
}

// SaveLastSentRequestCmd persists request (passed by value so the async
// closure captures its own copy, immune to whatever the caller's own
// *http.Request does afterward) as the request to restore on the app's
// next start (see storage.SaveLastSentRequest) - fire-and-forget (no
// result Msg): a failure here only means the *next* restart won't have
// something to restore, not something the user sending a request right
// now needs to be interrupted about.
func SaveLastSentRequestCmd(db *storage.SQLiteStorage, request http.Request) tea.Cmd {
	return func() tea.Msg {
		_ = db.SaveLastSentRequest(&request)
		return nil
	}
}

// ClearLastSentRequestCmd deletes whatever was saved by
// SaveLastSentRequestCmd (see storage.ClearLastSentRequest), so a reset
// request pane doesn't come back on the next app start - fire-and-forget,
// same rationale as SaveLastSentRequestCmd.
func ClearLastSentRequestCmd(db *storage.SQLiteStorage) tea.Cmd {
	return func() tea.Msg {
		_ = db.ClearLastSentRequest()
		return nil
	}
}

// SaveLastGroupFilterCmd persists name (see storage.SaveLastGroupFilter)
// as the sidebar's group filter to restore on the app's next start -
// fire-and-forget, same rationale as SaveLastSentRequestCmd/
// ClearLastSentRequestCmd: a failure here only affects what the *next*
// restart restores, not anything the user needs interrupted about right
// now.
func SaveLastGroupFilterCmd(db *storage.SQLiteStorage, name string) tea.Cmd {
	return func() tea.Msg {
		_ = db.SaveLastGroupFilter(name)
		return nil
	}
}

// SaveLastWSURLCmd persists url (see storage.SaveLastWSURL) as the
// WebSocket URL to restore on the app's next start - fire-and-forget,
// same rationale as SaveLastGroupFilterCmd. Callers only invoke this
// after a connection actually succeeds, never on a failed attempt.
func SaveLastWSURLCmd(db *storage.SQLiteStorage, url string) tea.Cmd {
	return func() tea.Msg {
		_ = db.SaveLastWSURL(url)
		return nil
	}
}

// SaveBodyStyleCmd persists name (see design.SaveBodyStyle) as the
// response pane's Body tab syntax highlighting style to restore on the
// app's next start - fire-and-forget, same rationale as
// SaveLastGroupFilterCmd: a failure here only affects what the *next*
// restart restores, not anything the user needs interrupted about right
// now. Saved to the same config.yaml the general UI theme selection lives
// in, not the database - see design.AppConfig.
func SaveBodyStyleCmd(name string) tea.Cmd {
	return func() tea.Msg {
		_ = design.SaveBodyStyle(name)
		return nil
	}
}

func LoadRequestsCmd(db *storage.SQLiteStorage) tea.Cmd {
	return func() tea.Msg {
		requests, err := db.Load()
		return RequestsLoadingMsg{
			Requests: requests,
			Err:      err,
		}
	}
}

func StartLoadTestCmd(config *http.JobConfig) tea.Cmd {
	return func() tea.Msg {
		return http.LoadTestStartMsg{Config: config}
	}
}

func WaitForLoadTestUpdatesCmd(updates <-chan *http.LoadTestStats, totalRequests int) tea.Cmd {
	return func() tea.Msg {
		stats, ok := <-updates
		if !ok || stats == nil {
			// Channel closed, test complete
			return http.LoadTestCompleteMsg{
				Stats:    nil,
				Duration: 0,
			}
		}

		if !stats.EndTime.IsZero() {
			return http.LoadTestCompleteMsg{
				Stats:    stats,
				Duration: stats.EndTime.Sub(stats.StartTime),
			}
		}

		progress := 0.0
		if totalRequests > 0 {
			progress = float64(stats.CompletedRequests) / float64(totalRequests)
		}

		return http.LoadTestStatsMsg{
			Stats:    stats,
			Progress: progress,
		}
	}
}
