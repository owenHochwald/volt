package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
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

		if stats.CompletedRequests >= stats.TotalRequests {
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
