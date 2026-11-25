package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RamXX/machinery/examples/crm/internal/fsm"
)

// TablesDir is where the runtime tables are read from:
// MACHINERY_TABLES_DIR when set (docs/formats.md section 7.3), else
// CRM_TABLES_DIR, else design/gen/tables.
func TablesDir() string {
	for _, v := range []string{"MACHINERY_TABLES_DIR", "CRM_TABLES_DIR"} {
		if dir := os.Getenv(v); dir != "" {
			return dir
		}
	}
	return filepath.Join("design", "gen", "tables")
}

// loadMachine reads one table and binds its units.
func loadMachine(dir, name string, u fsm.Units) (*fsm.Machine, error) {
	path := filepath.Join(dir, name+".table.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read table: %w", err)
	}
	m, err := fsm.Load(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := m.Bind(u); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return m, nil
}
