// Package gitcontrol supplies the closed subprocess environment used by
// repository-proof commands. Ambient Git variables can redirect object,
// worktree, config, index, namespace, hook, trace, or prompt behavior and are
// therefore never inherited.
package gitcontrol

import (
	"os"
	"strings"
)

// Environment removes every Git and locale override, then appends one fixed
// noninteractive, read-only, locale-stable configuration in canonical order.
func Environment(environ []string) []string {
	result := make([]string, 0, len(environ)+10)
	for _, item := range environ {
		key, _, _ := strings.Cut(item, "=")
		upper := strings.ToUpper(key)
		if strings.HasPrefix(upper, "GIT_") || upper == "LANG" || upper == "LANGUAGE" || strings.HasPrefix(upper, "LC_") {
			continue
		}
		result = append(result, item)
	}
	return append(result,
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_SYSTEM="+os.DevNull,
		"GIT_NO_LAZY_FETCH=1",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
		"LANG=C",
		"LC_ALL=C",
	)
}
