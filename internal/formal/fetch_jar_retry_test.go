package formal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFetchJarRetriesTransientFailures(t *testing.T) {
	for _, failure := range []string{"server error", "header EOF", "body EOF"} {
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			body := []byte("verified jar")
			sum := sha256.Sum256(body)
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if attempts.Add(1) <= 2 {
					switch failure {
					case "server error":
						http.Error(w, "temporary failure", http.StatusServiceUnavailable)
					case "header EOF":
						conn, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						_ = conn.Close()
					case "body EOF":
						w.Header().Set("Content-Length", "100")
						_, _ = w.Write([]byte("partial"))
					}
					return
				}
				w.Header().Set("Content-Length", fmt.Sprint(len(body)))
				_, _ = w.Write(body)
			}))
			defer server.Close()
			dest := filepath.Join(t.TempDir(), "tool.jar")
			got, err := fetchJar(dest, server.URL, "test jar", hex.EncodeToString(sum[:]))
			if err != nil || got != dest {
				t.Fatalf("fetch after transient failures = %q, %v", got, err)
			}
			if attempts.Load() != 3 {
				t.Fatalf("attempts = %d, want 3", attempts.Load())
			}
			cached, err := os.ReadFile(dest)
			if err != nil || !bytes.Equal(cached, body) {
				t.Fatalf("cached jar = %q, %v", cached, err)
			}
		})
	}
}

func TestFetchJarRetryFailurePolicy(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       int
		wantAttempts int32
		wantError    string
	}{
		{"bounded", http.StatusBadGateway, 3, "after 3 attempts"},
		{"permanent", http.StatusNotFound, 1, "after 1 attempt"},
		{"checksum after retry", http.StatusOK, 2, "checksum mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				attempt := attempts.Add(1)
				if tc.status == http.StatusOK && attempt == 1 {
					http.Error(w, "temporary", http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Length", "3")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte("bad"))
			}))
			defer server.Close()
			dir := t.TempDir()
			dest := filepath.Join(dir, "tool.jar")
			_, err := fetchJar(dest, server.URL, "test jar", strings.Repeat("0", 64))
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("fetch error = %v, want %q", err, tc.wantError)
			}
			if attempts.Load() != tc.wantAttempts {
				t.Fatalf("attempts = %d, want %d", attempts.Load(), tc.wantAttempts)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.Name() == "tool.jar" || strings.HasPrefix(entry.Name(), formalJarStagePrefix(dest)) {
					t.Fatalf("failed download retained %s", entry.Name())
				}
			}
		})
	}
}

func TestFetchJarRetryHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		http.Error(w, "temporary", http.StatusServiceUnavailable)
		cancel()
	}))
	defer server.Close()
	_, err := fetchJarContext(ctx, filepath.Join(t.TempDir(), "tool.jar"), server.URL, "test jar", strings.Repeat("0", 64))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
	if attempts.Load() != 1 {
		t.Fatalf("attempts after cancellation = %d", attempts.Load())
	}
}
