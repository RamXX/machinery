package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// brokenTables copies the committed tables and renames one action of the
// task table, so the service has no implementation for the new name.
func brokenTables(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files, err := filepath.Glob("../design/gen/tables/*.table.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no tables in design/gen/tables (%v)", err)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(f) == "task.table.json" {
			b = bytes.ReplaceAll(b, []byte(`"auditStart"`), []byte(`"auditStartRenamed"`))
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(f)), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// startFails runs a second instance with the given table variables and
// expects it to exit non-zero, naming the unbound action on stderr.
func startFails(t *testing.T, vars ...string) {
	t.Helper()
	addr, err := freeAddr()
	if err != nil {
		t.Fatal(err)
	}
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "MACHINERY_TABLES_DIR=") && !strings.HasPrefix(kv, "CRM_TABLES_DIR=") {
			env = append(env, kv)
		}
	}
	cmd := exec.Command(binPath)
	cmd.Dir = ".."
	cmd.Env = append(env, append([]string{"CRM_ADDR=" + addr, "CRM_DATABASE_URL=" + appURL, "CRM_ADMIN_DATABASE_URL=" + adminURL}, vars...)...)
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stderr, &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err = <-done:
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("crm with %v still running after 10s, want it to refuse to start", vars)
	}
	if err == nil {
		t.Fatalf("crm with %v exited 0, want non-zero", vars)
	}
	if !strings.Contains(stderr.String(), "auditStartRenamed") {
		t.Fatalf("crm with %v: stderr does not name the unbound action auditStartRenamed:\n%s", vars, stderr.String())
	}
}

func TestStartRefusesUnboundUnits(t *testing.T) {
	startFails(t, "CRM_TABLES_DIR="+brokenTables(t))
}

// TestMachineryTablesDirWins: MACHINERY_TABLES_DIR overrides CRM_TABLES_DIR
// (docs/formats.md section 7.3), so a broken table there stops the service
// even when CRM_TABLES_DIR names the good tables.
func TestMachineryTablesDirWins(t *testing.T) {
	good, err := filepath.Abs("../design/gen/tables")
	if err != nil {
		t.Fatal(err)
	}
	startFails(t, "CRM_TABLES_DIR="+good, "MACHINERY_TABLES_DIR="+brokenTables(t))
}
