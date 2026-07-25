package ui

import (
	"testing"
	"time"

	"github.com/owenHochwald/Volt/internal/http"
)

func TestWaitForLoadTestUpdatesUsesFinalMarker(t *testing.T) {
	tests := []struct {
		name  string
		stats *http.LoadTestStats
		want  any
	}{
		{
			name: "canceled final snapshot",
			stats: &http.LoadTestStats{
				StartTime:         time.Now().Add(-time.Second),
				EndTime:           time.Now(),
				TotalRequests:     100,
				CompletedRequests: 12,
			},
			want: http.LoadTestCompleteMsg{},
		},
		{
			name: "fully counted but not finalized",
			stats: &http.LoadTestStats{
				StartTime:         time.Now(),
				TotalRequests:     100,
				CompletedRequests: 100,
			},
			want: http.LoadTestStatsMsg{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates := make(chan *http.LoadTestStats, 1)
			updates <- tt.stats

			msg := WaitForLoadTestUpdatesCmd(updates, tt.stats.TotalRequests)()

			switch tt.want.(type) {
			case http.LoadTestCompleteMsg:
				complete, ok := msg.(http.LoadTestCompleteMsg)
				if !ok {
					t.Fatalf("message type = %T, want LoadTestCompleteMsg", msg)
				}
				if complete.Stats != tt.stats {
					t.Fatal("complete message did not preserve final statistics")
				}
			case http.LoadTestStatsMsg:
				if _, ok := msg.(http.LoadTestStatsMsg); !ok {
					t.Fatalf("message type = %T, want LoadTestStatsMsg", msg)
				}
			}
		})
	}
}
