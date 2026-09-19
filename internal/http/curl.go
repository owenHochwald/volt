package http

import (
	"sort"
	"strings"
)

// BuildCurlCommand renders req as a shell-ready curl command reproducing
// it exactly - call this on an already-resolved request (e.g.
// requestpane.RequestPane.resolvedRequest's result: path/query params
// substituted, server headers/variables merged in, absolute URL), not the
// raw stored template with its {name}/{{var}} placeholders still in it,
// since a pasted-and-run curl command has no template engine of its own
// to fill those in.
func BuildCurlCommand(req *Request) string {
	var b strings.Builder
	b.WriteString("curl -X ")
	b.WriteString(req.Method)
	b.WriteString(" \\\n  ")
	b.WriteString(shellQuote(req.URL))

	names := make([]string, 0, len(req.Headers))
	for name := range req.Headers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		b.WriteString(" \\\n  -H ")
		b.WriteString(shellQuote(name + ": " + req.Headers[name]))
	}

	if req.Body != "" {
		b.WriteString(" \\\n  --data-raw ")
		b.WriteString(shellQuote(req.Body))
	}
	return b.String()
}

// shellQuote wraps s in single quotes for a POSIX shell, escaping any
// literal single quote as '\” (close the quote, an escaped literal
// quote, reopen the quote) - the standard technique for safely embedding
// arbitrary text (a header value or body a user typed, which may contain
// anything - backticks, "$(...)", double quotes, etc.) inside a
// single-quoted shell argument without it being interpreted.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
