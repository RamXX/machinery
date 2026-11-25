// Package e2e runs the real crm binary over real HTTP against the real
// Postgres started by make up. Nothing is mocked.
package e2e

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	baseURL   string        // the service under test
	adminPool *pgxpool.Pool // the database as the admin role
	binPath   string        // the built service binary
	appURL    = env("CRM_DATABASE_URL", "postgres://crm_app:crm_app@127.0.0.1:15432/crm?sslmode=disable")
	adminURL  = env("CRM_ADMIN_DATABASE_URL", "postgres://crm_admin:crm_admin@127.0.0.1:15432/crm?sslmode=disable")
)

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()
	pool, err := connect(ctx, adminURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database not reachable: run make up (%v)\n", err)
		return 1
	}
	defer pool.Close()
	adminPool = pool
	dir, err := os.MkdirTemp("", "crm-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)
	binPath = filepath.Join(dir, "crm")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/crm")
	build.Dir = ".."
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build crm: %v\n", err)
		return 1
	}
	svc, err := start(appURL, adminURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	baseURL = svc.url
	if err := waitStatus(baseURL+"/healthz", http.StatusOK, 15*time.Second); err != nil {
		svc.stop()
		fmt.Fprintf(os.Stderr, "service not healthy: %v\n", err)
		return 1
	}
	code := m.Run()
	if err := svc.stop(); err != nil {
		fmt.Fprintf(os.Stderr, "service did not stop cleanly: %v\n", err)
		return 1
	}
	return code
}

func connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// service is one running instance of the binary.
type service struct {
	url  string
	cmd  *exec.Cmd
	done chan error
}

// start runs the binary on a free local port with the given database URLs.
func start(app, admin string) (*service, error) {
	addr, err := freeAddr()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(binPath)
	cmd.Dir = ".." // the project directory, where design/gen/tables is
	cmd.Env = append(os.Environ(), "CRM_ADDR="+addr, "CRM_DATABASE_URL="+app, "CRM_ADMIN_DATABASE_URL="+admin)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start crm: %w", err)
	}
	s := &service{url: "http://" + addr, cmd: cmd, done: make(chan error, 1)}
	go func() { s.done <- cmd.Wait() }()
	return s, nil
}

// stop sends SIGTERM and expects a clean exit within five seconds.
func (s *service) stop() error {
	_ = s.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case err := <-s.done:
		return err
	case <-time.After(5 * time.Second):
		_ = s.cmd.Process.Kill()
		return fmt.Errorf("no exit within 5s of SIGTERM")
	}
}

func freeAddr() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	return l.Addr().String(), nil
}

// waitStatus polls url until it answers with the wanted status.
func waitStatus(url string, want int, limit time.Duration) error {
	deadline := time.Now().Add(limit)
	last := "no answer"
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == want {
				return nil
			}
			last = fmt.Sprintf("%d %s", resp.StatusCode, body)
		} else {
			last = err.Error()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("%s did not answer %d within %s: %s", url, want, limit, last)
}
