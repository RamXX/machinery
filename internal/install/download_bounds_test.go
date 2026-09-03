package install

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateDownloadContentLengthRejectsInvalidAndOversize(t *testing.T) {
	policy := downloadPolicy{label: "test artifact", maxBytes: 4}
	for _, tc := range []struct {
		name   string
		length int64
		want   string
	}{
		{name: "negative", length: -2, want: "negative Content-Length"},
		{name: "oversize", length: 5, want: "exceeds 4-byte bound"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateDownloadContentLength(tc.length, policy); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Content-Length %d error = %v", tc.length, err)
			}
		})
	}
	for _, length := range []int64{-1, 0, 4} {
		if err := validateDownloadContentLength(length, policy); err != nil {
			t.Fatalf("safe Content-Length %d rejected: %v", length, err)
		}
	}
}

func TestDownloadBoundsDeclaredAndChunkedBodies(t *testing.T) {
	policy := downloadPolicy{label: "test artifact", maxBytes: 4}
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
		want    string
	}{
		{
			name: "missing length remains stream bounded",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.(http.Flusher).Flush()
				_, _ = w.Write([]byte("safe"))
			},
			want: "safe",
		},
		{
			name: "chunked overflow",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.(http.Flusher).Flush()
				_, _ = w.Write([]byte("large"))
			},
			wantErr: "stream exceeds 4-byte bound",
		},
		{
			name: "declared overflow",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "5")
				_, _ = w.Write([]byte("large"))
			},
			wantErr: "Content-Length 5 exceeds 4-byte bound",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handler)
			defer server.Close()
			destination := filepath.Join(t.TempDir(), "artifact")
			err := download(server.URL, destination, policy)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("download error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(destination)
			if err != nil || string(raw) != tc.want {
				t.Fatalf("download = %q, %v; want %q", raw, err, tc.want)
			}
		})
	}
}

func TestDownloadRejectsMalformedAndNegativeWireContentLength(t *testing.T) {
	for _, value := range []string{"invalid", "-2"} {
		t.Run(value, func(t *testing.T) {
			listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan error, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				request, err := http.ReadRequest(bufio.NewReader(conn))
				if err == nil {
					err = request.Body.Close()
				}
				if err == nil {
					_, err = fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Length: %s\r\nConnection: close\r\n\r\nsafe", value)
				}
				done <- err
			}()
			policy := downloadPolicy{label: "test artifact", maxBytes: 4}
			err = download("http://"+listener.Addr().String(), filepath.Join(t.TempDir(), "artifact"), policy)
			if err == nil || !strings.Contains(err.Error(), "request failed") {
				t.Fatalf("wire Content-Length %q error = %v", value, err)
			}
			if serveErr := <-done; serveErr != nil {
				t.Fatal(serveErr)
			}
		})
	}
}

func TestDownloadEnforcesEveryArtifactPolicyContentLength(t *testing.T) {
	for _, policy := range []downloadPolicy{releaseAPIDownload, releaseChecksumsDownload, releaseSourceDownload, releaseBinaryDownload} {
		t.Run(policy.label, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", policy.maxBytes+1))
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			err := download(server.URL, filepath.Join(t.TempDir(), "artifact"), policy)
			if err == nil || !strings.Contains(err.Error(), "Content-Length") || !strings.Contains(err.Error(), "exceeds") {
				t.Fatalf("%s oversized declaration error = %v", policy.label, err)
			}
		})
	}
}

func TestDownloadedFileReadersRejectMissingAndSparseOversize(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := readDownloadedFile(missing, releaseAPIDownload); err == nil || !os.IsNotExist(err) {
		t.Fatalf("missing bounded read error = %v", err)
	}
	if _, err := hashDownloadedFile(missing, releaseBinaryDownload); err == nil || !os.IsNotExist(err) {
		t.Fatalf("missing bounded hash error = %v", err)
	}

	for _, policy := range []downloadPolicy{releaseChecksumsDownload, releaseSourceDownload, releaseBinaryDownload} {
		t.Run(policy.label, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sparse")
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.Truncate(policy.maxBytes + 1); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if _, err := readDownloadedFile(path, policy); err == nil || !strings.Contains(err.Error(), "invalid size") {
				t.Fatalf("sparse bounded read error = %v", err)
			}
			if _, err := hashDownloadedFile(path, policy); err == nil || !strings.Contains(err.Error(), "invalid size") {
				t.Fatalf("sparse bounded hash error = %v", err)
			}
		})
	}
}

func TestReserveSourceArchiveMemberRejectsExpansionBounds(t *testing.T) {
	for _, tc := range []struct {
		name        string
		size, total int64
		want        string
	}{
		{name: "negative member", size: -1, want: "invalid size"},
		{name: "oversized member", size: releaseSourceMemberMaxBytes + 1, want: "invalid size"},
		{name: "negative total", total: -1, want: "invalid accumulated size"},
		{name: "oversized total", total: releaseSourceTreeMaxBytes + 1, want: "invalid accumulated size"},
		{name: "aggregate overflow", size: 2, total: releaseSourceTreeMaxBytes - 1, want: "extracted byte bound"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := reserveSourceArchiveMember("machinery/member", tc.size, tc.total); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("reserve archive size=%d total=%d error = %v", tc.size, tc.total, err)
			}
		})
	}
}
