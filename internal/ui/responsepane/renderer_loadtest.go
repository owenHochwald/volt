package responsepane

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/owenHochwald/Volt/internal/apperror"
	"github.com/owenHochwald/Volt/internal/http"
)

// renderLoadTestOverview renders the overview tab for load test results
func (m ResponsePane) renderLoadTestOverview() string {
	stats := m.LoadTestStats
	if stats == nil {
		return "No data"
	}

	successCount := stats.CompletedRequests - stats.FailedRequests
	successRate := 0.0
	if stats.CompletedRequests > 0 {
		successRate = float64(successCount) / float64(stats.CompletedRequests) * 100
	}
	failureRate := 100 - successRate
	if stats.CompletedRequests == 0 {
		failureRate = 0
	}
	elapsed := loadTestElapsed(stats)
	throughput := 0.0
	if elapsed.Seconds() > 0 {
		throughput = float64(stats.CompletedRequests) / elapsed.Seconds()
	}
	p50 := time.Duration(0)
	if stats.Percentiles != nil {
		p50 = stats.Percentiles.Percentile(50)
	}

	var b strings.Builder
	b.WriteString(m.styles.Metric.Label.Render("PERFORMANCE"))
	b.WriteString("\n")
	b.WriteString(m.styles.Metric.Value.Render(formatThroughput(throughput)))
	b.WriteString(m.styles.Metric.Unit.Render(" req/s"))
	b.WriteString("       ")
	b.WriteString(m.styles.Metric.Value.Render(formatLatency(p50)))
	b.WriteString(m.styles.Metric.Unit.Render(" p50"))
	b.WriteString("       ")
	b.WriteString(m.styles.Metric.Value.Render(fmt.Sprintf("%.2f%%", failureRate)))
	b.WriteString(m.styles.Metric.Unit.Render(" errors"))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Metric.Label.Render("LATENCY CURRENT"))
	b.WriteString("\n")
	b.WriteString(m.renderLatencySparkline())
	b.WriteString("\n\n")

	b.WriteString(m.styles.Metric.Label.Render("OUTCOME"))
	b.WriteString("\n")
	b.WriteString(m.styles.Text.ResponseLabel.Render("Success"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(fmt.Sprintf(
		"%s (%.1f%%)",
		formatRequestCount(successCount),
		successRate,
	)))
	b.WriteString("    ")
	b.WriteString(m.styles.Text.ResponseLabel.Render("Failed"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(fmt.Sprintf(
		"%s (%.1f%%)",
		formatRequestCount(stats.FailedRequests),
		failureRate,
	)))
	b.WriteString("    ")
	b.WriteString(m.styles.Text.ResponseLabel.Render("Duration"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(elapsed.Round(time.Millisecond).String()))

	return b.String()
}

func (m ResponsePane) renderLatencySparkline() string {
	if len(m.latencySamples) == 0 {
		return m.styles.Text.Muted.Render("waiting for samples…")
	}

	width := min(max(m.width-2, 8), maxLatencySamples)
	samples := m.latencySamples
	if len(samples) > width {
		samples = samples[len(samples)-width:]
	}

	minimum, maximum := samples[0], samples[0]
	for _, sample := range samples[1:] {
		minimum = min(minimum, sample)
		maximum = max(maximum, sample)
	}

	const levels = "▁▂▃▄▅▆▇█"
	var b strings.Builder
	for _, sample := range samples {
		index := 3
		if maximum > minimum {
			ratio := float64(sample-minimum) / float64(maximum-minimum)
			index = int(math.Round(ratio * 7))
		}
		b.WriteRune([]rune(levels)[index])
	}
	return m.styles.Chart.Secondary.Render(b.String())
}

func loadTestProgress(stats *http.LoadTestStats) float64 {
	if stats == nil || stats.TotalRequests <= 0 {
		return 0
	}
	return min(max(
		float64(stats.CompletedRequests)/float64(stats.TotalRequests),
		0,
	), 1)
}

func loadTestElapsed(stats *http.LoadTestStats) time.Duration {
	if stats == nil || stats.StartTime.IsZero() {
		return 0
	}
	end := time.Now()
	if !stats.EndTime.IsZero() {
		end = stats.EndTime
	}
	return max(end.Sub(stats.StartTime), 0)
}

func formatRequestCount(value int) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	digits := strconv.Itoa(value)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return sign + digits
}

func formatThroughput(value float64) string {
	rounded := int64(math.Round(value))
	return formatRequestCount(int(rounded))
}

func formatLatency(value time.Duration) string {
	if value <= 0 {
		return "—"
	}
	if value < time.Millisecond {
		return fmt.Sprintf("%.0fµs", float64(value)/float64(time.Microsecond))
	}
	return fmt.Sprintf("%.1fms", float64(value)/float64(time.Millisecond))
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
