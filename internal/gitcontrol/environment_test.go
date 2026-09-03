package gitcontrol

import (
	"os"
	"strings"
	"testing"
)

func TestEnvironmentStripsAmbientGitAndLocaleInjection(t *testing.T) {
	got := Environment([]string{
		"PATH=/bin", "GIT_DIR=/outside", "git_work_tree=/other", "GIT_TRACE=1",
		"GIT_CONFIG_COUNT=1", "LC_MESSAGES=tr_TR", "LC_ALL=ja_JP", "LANG=de_DE", "LANGUAGE=fr",
	})
	joined := "\n" + strings.Join(got, "\n") + "\n"
	for _, forbidden := range []string{"GIT_DIR=/outside", "git_work_tree=/other", "GIT_TRACE=1", "GIT_CONFIG_COUNT=1", "LC_MESSAGES=tr_TR", "LC_ALL=ja_JP", "LANG=de_DE", "LANGUAGE=fr"} {
		if strings.Contains(joined, "\n"+forbidden+"\n") {
			t.Errorf("ambient override survived: %s", forbidden)
		}
	}
	for _, required := range []string{"PATH=/bin", "GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CONFIG_SYSTEM=" + os.DevNull, "GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C", "LANG=C"} {
		if !strings.Contains(joined, "\n"+required+"\n") {
			t.Errorf("closed environment lacks %s: %v", required, got)
		}
	}
}
