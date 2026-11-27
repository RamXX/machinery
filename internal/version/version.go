// Package version holds the machinery version.
package version

import "runtime/debug"

// Version and BuildDate may be set at link time by the Makefile.
var (
	Version   = "dev"
	BuildDate = "unknown"
)

// Commit reports the source revision embedded by the Go toolchain.
func Commit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	var revision string
	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision == "" {
		return "unknown"
	}
	if modified {
		revision += "-dirty"
	}
	return revision
}

// Pinned upstream versions the formats are designed against (docs/formats.md).
const (
	ModelithMin = "0.6.0"
	XStateMin   = "5.33.4"
	TLAVersion  = "v1.7.4"
	TLASHA256   = "936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88"
	AlloyVer    = "v6.2.0"
	AlloySHA256 = "6b8c1cb5bc93bedfc7c61435c4e1ab6e688a242dc702a394628d9a9801edb78d"
)
