// Package difftext computes a line-based diff between two texts - the
// response pane's "diff two responses" feature (see responsepane's
// diff.go) is what actually uses this, but the algorithm itself has
// nothing response-specific about it.
package difftext

// Op identifies what a Line represents in the transformation from the
// first text to the second.
type Op int

const (
	Equal Op = iota
	Insert
	Delete
)

// Line is one line of the diff output, tagged with how it relates to the
// two inputs: unchanged (Equal, present in both), added (Insert, only in
// b), or removed (Delete, only in a).
type Line struct {
	Op   Op
	Text string
}

// maxCells bounds the classic O(n*m) LCS table this uses - a and b
// beyond it (e.g. two very large, almost entirely different response
// bodies) would need gigabytes of memory for a table neither response
// pane nor a terminal could usefully show anyway, so Lines falls back to
// ByLine (mark all of a as removed, all of b as added - still a
// correct, if unhelpful, diff) rather than risk exhausting memory.
const maxCells = 4_000_000

// Lines computes a line-based diff of a and b via the classic
// longest-common-subsequence dynamic-programming algorithm, returning
// the sequence of Equal/Insert/Delete operations that transforms a into
// b - the smallest such sequence, so unchanged lines (typically most of
// a response) show up as Equal instead of a spurious delete+insert pair.
func Lines(a, b []string) []Line {
	n, m := len(a), len(b)
	if n*m > maxCells {
		return byLine(a, b)
	}

	// dp[i][j] = length of the LCS of a[i:] and b[j:]. Sized (n+1)x(m+1)
	// so dp[n][*] and dp[*][m] (one past the end of each slice) are valid
	// zero-value lookups, the base case every other cell is built from.
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	lines := make([]Line, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			lines = append(lines, Line{Op: Equal, Text: a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			lines = append(lines, Line{Op: Delete, Text: a[i]})
			i++
		default:
			lines = append(lines, Line{Op: Insert, Text: b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		lines = append(lines, Line{Op: Delete, Text: a[i]})
	}
	for ; j < m; j++ {
		lines = append(lines, Line{Op: Insert, Text: b[j]})
	}
	return lines
}

// byLine is Lines' fallback for inputs too large for the LCS table -
// every line of a is a deletion, every line of b an insertion. Still a
// technically valid diff (concatenating a's deletions with b's
// insertions does transform a into b), just not a minimal one.
func byLine(a, b []string) []Line {
	lines := make([]Line, 0, len(a)+len(b))
	for _, l := range a {
		lines = append(lines, Line{Op: Delete, Text: l})
	}
	for _, l := range b {
		lines = append(lines, Line{Op: Insert, Text: l})
	}
	return lines
}
