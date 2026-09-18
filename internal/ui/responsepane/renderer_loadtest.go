package responsepane

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/owenHochwald/Volt/internal/apperror"
)

// renderLoadTestOverview renders the overview tab for load test results
func (m ResponsePane) renderLoadTestOverview() string {
	stats := m.LoadTestStats
	if stats == nil {
		return "No data"
	}

	var b strings.Builder
	b.WriteString("Load Test Results\n")
	b.WriteString(strings.Repeat("─", 60) + "\n\n")

	// Calculate success metrics
	successCount := stats.CompletedRequests - stats.FailedRequests
	successRate := 0.0
	if stats.CompletedRequests > 0 {
		successRate = float64(successCount) / float64(stats.CompletedRequests) * 100
	}

	// Requests
	b.WriteString(m.styles.Text.ResponseLabel.Render("Requests"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(fmt.Sprintf("%d / %d", stats.CompletedRequests, stats.TotalRequests)))
	b.WriteString("\n\n")

	// Success
	b.WriteString(m.styles.Text.ResponseLabel.Render("Success"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(fmt.Sprintf("%d (%.1f%%)", successCount, successRate)))
	b.WriteString("\n\n")

	// Failed
	b.WriteString(m.styles.Text.ResponseLabel.Render("Failed"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(fmt.Sprintf("%d (%.1f%%)", stats.FailedRequests, 100-successRate)))
	b.WriteString("\n\n")

	// Calculate throughput
	elapsed := time.Since(stats.StartTime)
	if !stats.EndTime.IsZero() {
		elapsed = stats.EndTime.Sub(stats.StartTime)
	}
	throughput := 0.0
	if elapsed.Seconds() > 0 {
		throughput = float64(stats.CompletedRequests) / elapsed.Seconds()
	}

	// Throughput
	b.WriteString(m.styles.Text.ResponseLabel.Render("Throughput"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(fmt.Sprintf("%.1f req/s", throughput)))
	b.WriteString("\n\n")

	// Duration
	b.WriteString(m.styles.Text.ResponseLabel.Render("Duration"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(elapsed.Round(time.Millisecond).String()))
	b.WriteString("\n")

	return b.String()
}

// renderLoadTestLatency renders the latency distribution tab for load test results
func (m ResponsePane) renderLoadTestLatency() string {
	stats := m.LoadTestStats
	if stats == nil || stats.Percentiles == nil {
		return "No latency data"
	}

	var b strings.Builder
	b.WriteString("Latency Distribution\n")
	b.WriteString(strings.Repeat("─", 60) + "\n\n")

	// Min
	b.WriteString(m.styles.Text.ResponseLabel.Render("Min"))
	b.WriteString(":    ")
	b.WriteString(m.styles.Text.Value.Render(stats.MinDuration.Round(time.Millisecond).String()))
	b.WriteString("\n\n")

	// p50
	b.WriteString(m.styles.Text.ResponseLabel.Render("p50"))
	b.WriteString(":    ")
	b.WriteString(m.styles.Text.Value.Render(stats.Percentiles.Percentile(50).Round(time.Millisecond).String()))
	b.WriteString("\n\n")

	// p90
	b.WriteString(m.styles.Text.ResponseLabel.Render("p90"))
	b.WriteString(":    ")
	b.WriteString(m.styles.Text.Value.Render(stats.Percentiles.Percentile(90).Round(time.Millisecond).String()))
	b.WriteString("\n\n")

	// p95
	b.WriteString(m.styles.Text.ResponseLabel.Render("p95"))
	b.WriteString(":    ")
	b.WriteString(m.styles.Text.Value.Render(stats.Percentiles.Percentile(95).Round(time.Millisecond).String()))
	b.WriteString("\n\n")

	// p99
	b.WriteString(m.styles.Text.ResponseLabel.Render("p99"))
	b.WriteString(":    ")
	b.WriteString(m.styles.Text.Value.Render(stats.Percentiles.Percentile(99).Round(time.Millisecond).String()))
	b.WriteString("\n\n")

	// Max
	b.WriteString(m.styles.Text.ResponseLabel.Render("Max"))
	b.WriteString(":    ")
	b.WriteString(m.styles.Text.Value.Render(stats.MaxDuration.Round(time.Millisecond).String()))
	b.WriteString("\n")

	return b.String()
}

// renderLoadTestErrors renders the error breakdown tab for load test results
func (m ResponsePane) renderLoadTestErrors() string {
	stats := m.LoadTestStats
	if stats == nil {
		return "No error data"
	}

	var b strings.Builder
	b.WriteString("Error Breakdown\n")
	b.WriteString(strings.Repeat("─", 60) + "\n\n")

	if len(stats.Errors) == 0 {
		b.WriteString(m.styles.Text.Value.Render("No errors encountered!"))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Text.Faint.Render("All requests completed successfully."))
		b.WriteString("\n\n")
	} else {
		classes := make([]string, 0, len(stats.Errors))
		for class := range stats.Errors {
			classes = append(classes, class)
		}
		sort.Strings(classes)
		for _, class := range classes {
			b.WriteString(m.styles.Text.ResponseKey.Render(apperror.ErrorClassLabel(class)))
			b.WriteString(": ")
			b.WriteString(m.styles.Text.Value.Render(fmt.Sprintf("%d occurrences", stats.Errors[class])))
			b.WriteString("\n\n")
		}
	}

	if len(stats.StatusCodes) > 0 {
		b.WriteString(m.styles.Text.ResponseLabel.Render("HTTP Status Codes"))
		b.WriteString("\n")
		statuses := make([]int, 0, len(stats.StatusCodes))
		for status := range stats.StatusCodes {
			statuses = append(statuses, status)
		}
		sort.Ints(statuses)
		for _, status := range statuses {
			b.WriteString(m.styles.Text.ResponseKey.Render(fmt.Sprintf("%d", status)))
			b.WriteString(": ")
			b.WriteString(m.styles.Text.Value.Render(fmt.Sprintf("%d responses", stats.StatusCodes[status])))
			b.WriteString("\n")
		}
	}

	return b.String()
}
