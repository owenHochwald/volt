package buildinfo

import (
	"runtime/debug"
	"strings"
)

var version = "dev"

// Version returns the release version embedded by GoReleaser or Go's module
// metadata. Local source builds are labeled dev instead of claiming a release.
func Version() string {
	if normalized := normalizeVersion(version); normalized != "dev" {
		return normalized
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		return normalizeVersion(info.Main.Version)
	}
	return "dev"
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "", "(devel)", "dev":
		return "dev"
	}
	if strings.HasPrefix(value, "v") {
		return value
	}
	return "v" + value
}
