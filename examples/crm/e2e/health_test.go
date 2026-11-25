package e2e

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type body struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func getHealth(t *testing.T, url string) (int, body) {
	t.Helper()
	resp, err := http.Get(url + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	var b body
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		t.Fatalf("decode /healthz body: %v", err)
	}
	return resp.StatusCode, b
}

func TestHealthOK(t *testing.T) {
	code, b := getHealth(t, baseURL)
	if code != http.StatusOK || b.Status != "ok" {
		t.Fatalf("GET /healthz = %d %+v, want 200 status ok", code, b)
	}
}

func TestHealthStoreDown(t *testing.T) {
	dead, err := freeAddr() // closed again on return: nothing listens there
	if err != nil {
		t.Fatal(err)
	}
	url := "postgres://crm_app:crm_app@" + dead + "/crm?sslmode=disable&connect_timeout=1"
	svc, err := start(url, url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := svc.stop(); err != nil {
			t.Errorf("second instance did not stop cleanly: %v", err)
		}
	}()
	if err := waitListening(svc.url[len("http://"):], 10*time.Second); err != nil {
		t.Fatal(err)
	}
	began := time.Now()
	code, b := getHealth(t, svc.url)
	took := time.Since(began)
	if code != http.StatusServiceUnavailable || b.Code != "STORE_UNAVAILABLE" || b.Message == "" {
		t.Fatalf("GET /healthz with the database down = %d %+v, want 503 STORE_UNAVAILABLE with a message", code, b)
	}
	if took > 3*time.Second {
		t.Fatalf("GET /healthz with the database down took %s, want at most 3s", took)
	}
}

func waitListening(addr string, limit time.Duration) error {
	deadline := time.Now().Add(limit)
	for {
		c, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			return c.Close()
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestMigrationsApplied(t *testing.T) {
	ctx := context.Background()
	var rows int
	if err := adminPool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&rows); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if rows != 1 {
		t.Fatalf("schema_migrations has %d rows, want 1", rows)
	}
	var exists bool
	if err := adminPool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'crm_app')").Scan(&exists); err != nil {
		t.Fatalf("look up role crm_app: %v", err)
	}
	if !exists {
		t.Fatal("role crm_app does not exist")
	}
}

func TestAppRoleIsNotSuperuser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, appURL)
	if err != nil {
		t.Fatalf("connect as crm_app: %v", err)
	}
	defer conn.Close(ctx)
	var su string
	if err := conn.QueryRow(ctx, "SHOW is_superuser").Scan(&su); err != nil {
		t.Fatalf("SHOW is_superuser: %v", err)
	}
	if su != "off" {
		t.Fatalf("crm_app is_superuser = %q, want off", su)
	}
}
