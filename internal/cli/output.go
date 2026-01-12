package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/owenHochwald/Volt/internal/http"
)

// FormatOutput writes results in requested format
func FormatOutput(stats *http.LoadTestStats, config *BenchConfig) error {
	var output string

	if config.JSON {
		output = formatJSON(stats)
	} else if config.Quiet {
		output = formatQuiet(stats)
	} else {
		output = formatTable(stats)
	}

	// Write to file or stdout
	if config.Output != "" {
		return os.WriteFile(config.Output, []byte(output), 0644)
	}

	fmt.Print(output)
	return nil
}

// formatTable produces human-readable table output
func formatTable(stats *http.LoadTestStats) string {
	duration := stats.EndTime.Sub(stats.StartTime)
	successRate := float64(stats.CompletedRequests-stats.FailedRequests) / float64(stats.CompletedRequests) * 100
	rps := float64(stats.CompletedRequests) / duration.Seconds()

	var out strings.Builder

	out.WriteString("\nVolt Load Test Results\n\n")

	out.WriteString(fmt.Sprintf("Duration:       %.2fs\n", duration.Seconds()))
	out.WriteString(fmt.Sprintf("Total Requests: %s\n\n", formatNumber(stats.CompletedRequests)))

	out.WriteString("Summary:\n")
	out.WriteString(fmt.Sprintf("  Success:      %s (%.2f%%)\n",
		formatNumber(stats.CompletedRequests-stats.FailedRequests), successRate))
	out.WriteString(fmt.Sprintf("  Failed:       %s (%.2f%%)\n",
		formatNumber(stats.FailedRequests), 100-successRate))
	out.WriteString(fmt.Sprintf("  Requests/sec: %.2f\n", rps))

	// Calculate data transfer rate
	totalBytes := stats.BytesSent + stats.BytesRecv
	dataSec := float64(totalBytes) / duration.Seconds() / (1024 * 1024) // MB/s
	out.WriteString(fmt.Sprintf("  Data/sec:     %.2f MB\n\n", dataSec))

	out.WriteString("Latency:\n")
	out.WriteString(fmt.Sprintf("  Min:          %s\n", formatDuration(stats.MinDuration)))

	avgDuration := stats.TotalDuration / time.Duration(stats.CompletedRequests)
	out.WriteString(fmt.Sprintf("  Mean:         %s\n", formatDuration(avgDuration)))
	out.WriteString(fmt.Sprintf("  p50:          %s\n", formatDuration(stats.Percentiles.Percentile(50))))
	out.WriteString(fmt.Sprintf("  p95:          %s\n", formatDuration(stats.Percentiles.Percentile(95))))
	out.WriteString(fmt.Sprintf("  p99:          %s\n", formatDuration(stats.Percentiles.Percentile(99))))
	out.WriteString(fmt.Sprintf("  Max:          %s\n\n", formatDuration(stats.MaxDuration)))

	if len(stats.Errors) > 0 {
		out.WriteString("Status Codes:\n")
		for code, count := range stats.Errors {
			out.WriteString(fmt.Sprintf("  %s:          %s\n", code, formatNumber(int(count))))
		}
	}

	return out.String()
}

// formatJSON produces machine-readable JSON output
func formatJSON(stats *http.LoadTestStats) string {
	duration := stats.EndTime.Sub(stats.StartTime)

	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"totalRequests":     stats.TotalRequests,
			"completedRequests": stats.CompletedRequests,
			"failedRequests":    stats.FailedRequests,
			"successRate":       float64(stats.CompletedRequests-stats.FailedRequests) / float64(stats.CompletedRequests),
			"throughput":        float64(stats.CompletedRequests) / duration.Seconds(),
			"durationMs":        duration.Milliseconds(),
		},
		"latency": map[string]interface{}{
			"minMs": stats.MinDuration.Milliseconds(),
			"avgMs": (stats.TotalDuration / time.Duration(stats.CompletedRequests)).Milliseconds(),
			"p50Ms": stats.Percentiles.Percentile(50).Milliseconds(),
			"p90Ms": stats.Percentiles.Percentile(90).Milliseconds(),
			"p95Ms": stats.Percentiles.Percentile(95).Milliseconds(),
			"p99Ms": stats.Percentiles.Percentile(99).Milliseconds(),
			"maxMs": stats.MaxDuration.Milliseconds(),
		},
		"errors": stats.Errors,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data) + "\n"
}

// formatQuiet produces one-line summary
func formatQuiet(stats *http.LoadTestStats) string {
	duration := stats.EndTime.Sub(stats.StartTime)
	rps := float64(stats.CompletedRequests) / duration.Seconds()
	p50 := stats.Percentiles.Percentile(50)
	p99 := stats.Percentiles.Percentile(99)

	return fmt.Sprintf("Requests: %d | RPS: %.2f | p50: %s | p99: %s | Failed: %d\n",
		stats.CompletedRequests, rps, formatDuration(p50), formatDuration(p99), stats.FailedRequests)
}

func formatNumber(n int) string {
	// Add thousand separators
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Insert commas
	var result []rune
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, c)
	}
	return string(result)
}

func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.0fµs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
