package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
	successRate := 0.0
	if stats.CompletedRequests > 0 {
		successRate = float64(stats.CompletedRequests-stats.FailedRequests) / float64(stats.CompletedRequests) * 100
	}
	rps := 0.0
	if duration > 0 {
		rps = float64(stats.CompletedRequests) / duration.Seconds()
	}

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
	dataSec := 0.0
	if duration > 0 {
		dataSec = float64(totalBytes) / duration.Seconds() / (1024 * 1024) // MB/s
	}
	out.WriteString(fmt.Sprintf("  Data/sec:     %.2f MB\n\n", dataSec))

	out.WriteString("Latency:\n")
	out.WriteString(fmt.Sprintf("  Min:          %s\n", formatDuration(stats.MinDuration)))

	out.WriteString(fmt.Sprintf("  Mean:         %s\n", formatDuration(stats.MeanDuration())))
	out.WriteString(fmt.Sprintf("  p50:          %s\n", formatDuration(stats.Percentiles.Percentile(50))))
	out.WriteString(fmt.Sprintf("  p95:          %s\n", formatDuration(stats.Percentiles.Percentile(95))))
	out.WriteString(fmt.Sprintf("  p99:          %s\n", formatDuration(stats.Percentiles.Percentile(99))))
	out.WriteString(fmt.Sprintf("  Max:          %s\n\n", formatDuration(stats.MaxDuration)))

	if len(stats.StatusCodes) > 0 {
		out.WriteString("Status Codes:\n")
		statuses := make([]int, 0, len(stats.StatusCodes))
		for status := range stats.StatusCodes {
			statuses = append(statuses, status)
		}
		sort.Ints(statuses)
		for _, status := range statuses {
			out.WriteString(fmt.Sprintf("  %d:          %s\n", status, formatNumber(int(stats.StatusCodes[status]))))
		}
	}
	if len(stats.Errors) > 0 {
		out.WriteString("Errors:\n")
		errorClasses := make([]string, 0, len(stats.Errors))
		for class := range stats.Errors {
			errorClasses = append(errorClasses, class)
		}
		sort.Strings(errorClasses)
		for _, class := range errorClasses {
			out.WriteString(fmt.Sprintf("  %s:          %s\n", class, formatNumber(int(stats.Errors[class]))))
		}
	}

	return out.String()
}

// formatJSON produces machine-readable JSON output
func formatJSON(stats *http.LoadTestStats) string {
	duration := stats.EndTime.Sub(stats.StartTime)
	successRate := 0.0
	throughput := 0.0
	if stats.CompletedRequests > 0 {
		successRate = float64(stats.CompletedRequests-stats.FailedRequests) / float64(stats.CompletedRequests)
	}
	if duration > 0 {
		throughput = float64(stats.CompletedRequests) / duration.Seconds()
	}

	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"totalRequests":     stats.TotalRequests,
			"completedRequests": stats.CompletedRequests,
			"failedRequests":    stats.FailedRequests,
			"successRate":       successRate,
			"throughput":        throughput,
			"durationMs":        durationMilliseconds(duration),
		},
		"latency": map[string]interface{}{
			"minMs": durationMilliseconds(stats.MinDuration),
			"avgMs": durationMilliseconds(stats.MeanDuration()),
			"p50Ms": durationMilliseconds(stats.Percentiles.Percentile(50)),
			"p90Ms": durationMilliseconds(stats.Percentiles.Percentile(90)),
			"p95Ms": durationMilliseconds(stats.Percentiles.Percentile(95)),
			"p99Ms": durationMilliseconds(stats.Percentiles.Percentile(99)),
			"maxMs": durationMilliseconds(stats.MaxDuration),
		},
		"errors":      stats.Errors,
		"statusCodes": stats.StatusCodes,
		"transfer": map[string]interface{}{
			"bytesSent":     stats.BytesSent,
			"bytesReceived": stats.BytesRecv,
		},
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data) + "\n"
}

// formatQuiet produces one-line summary
func formatQuiet(stats *http.LoadTestStats) string {
	duration := stats.EndTime.Sub(stats.StartTime)
	rps := 0.0
	if duration > 0 {
		rps = float64(stats.CompletedRequests) / duration.Seconds()
	}
	p50 := stats.Percentiles.Percentile(50)
	p99 := stats.Percentiles.Percentile(99)

	return fmt.Sprintf("Requests: %d | RPS: %.2f | p50: %s | p99: %s | Failed: %d\n",
		stats.CompletedRequests, rps, formatDuration(p50), formatDuration(p99), stats.FailedRequests)
}

func durationMilliseconds(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / float64(time.Millisecond)
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
