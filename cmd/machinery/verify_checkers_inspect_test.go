package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type ociInspectFixtureResponse struct {
	Platform string
	Digest   string
	Error    string
}

type ociInspectFixture struct {
	Selected   ociInspectFixtureResponse
	Unselected ociInspectFixtureResponse
}

// runCheckerOCIInspectFixture is a fake Docker CLI run by the existing process
// fixture. It records real argv and responds independently to selected and
// unselected inspection, including failures that must never trigger fallback.
func runCheckerOCIInspectFixture() {
	path := checkerProcessFixtureArgument("oci-inspect")
	var fixture ociInspectFixture
	if err := json.Unmarshal(mustReadProcessFile(path), &fixture); err != nil {
		os.Exit(40)
	}
	var args []string
	for i, arg := range os.Args {
		if arg == "oci-inspect" && i+2 < len(os.Args) {
			args = os.Args[i+2:]
			break
		}
	}
	log, err := os.OpenFile(path+".calls", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(41)
	}
	if err := json.NewEncoder(log).Encode(args); err != nil {
		os.Exit(42)
	}
	if err := log.Close(); err != nil {
		os.Exit(43)
	}
	response := fixture.Unselected
	format := ""
	for i, arg := range args {
		if arg == "--platform" {
			response = fixture.Selected
		}
		if arg == "--format" && i+1 < len(args) {
			format = args[i+1]
		}
	}
	if response.Error != "" {
		_, _ = fmt.Fprintln(os.Stderr, response.Error)
		os.Exit(1)
	}
	parts := strings.Split(response.Platform, "/")
	imageOS, arch, variant := "", "", ""
	if len(parts) >= 2 {
		imageOS, arch = parts[0], parts[1]
	}
	if len(parts) == 3 {
		variant = parts[2]
	}
	enc := json.NewEncoder(os.Stdout)
	for _, value := range []any{[]string{response.Digest}, imageOS, arch} {
		if err := enc.Encode(value); err != nil {
			os.Exit(44)
		}
	}
	if strings.Contains(format, ".Variant") {
		if err := enc.Encode(variant); err != nil {
			os.Exit(45)
		}
	}
	os.Exit(0)
}

func TestVerifyLocalOCIImagePlatformInspect(t *testing.T) {
	correct := ociInspectFixtureResponse{Platform: "linux/arm64", Digest: testRuntimeImage}
	wrongPlatform := ociInspectFixtureResponse{Platform: "linux/amd64", Digest: testRuntimeImage}
	wrongOS := ociInspectFixtureResponse{Platform: "windows/arm64", Digest: testRuntimeImage}
	wrongDigest := ociInspectFixtureResponse{Platform: "linux/arm64", Digest: "example.invalid/machinery/checker-test@sha256:" + strings.Repeat("f", 64)}
	emptyPlatform := ociInspectFixtureResponse{Digest: testRuntimeImage}
	variant := ociInspectFixtureResponse{Platform: "linux/arm/v7", Digest: testRuntimeImage}
	unknownFlag := ociInspectFixtureResponse{Error: "unknown flag: --platform"}
	oldAPI := ociInspectFixtureResponse{Error: "--platform requires API version 1.49, but the Docker daemon API version is 1.48"}
	cases := []struct {
		name, required, wantError string
		selected, unselected      ociInspectFixtureResponse
		fallback                  bool
	}{
		{name: "containerd_empty_unselected", selected: correct, unselected: emptyPlatform},
		{name: "default_arm64_variant", selected: ociInspectFixtureResponse{Platform: "linux/arm64/v8", Digest: testRuntimeImage}, unselected: emptyPlatform},
		{name: "default_amd64_variant", required: "linux/amd64", selected: ociInspectFixtureResponse{Platform: "linux/amd64/v1", Digest: testRuntimeImage}, unselected: emptyPlatform},
		{name: "nondefault_arm64_variant", selected: ociInspectFixtureResponse{Platform: "linux/arm64/v9", Digest: testRuntimeImage}, unselected: correct, wantError: "does not match required platform"},
		{name: "fallback_default_arm64_variant", selected: oldAPI, unselected: ociInspectFixtureResponse{Platform: "linux/arm64/v8", Digest: testRuntimeImage}, fallback: true},
		{name: "selected_wrong_architecture", selected: wrongPlatform, unselected: correct, wantError: "platform linux/amd64 does not match required platform linux/arm64"},
		{name: "selected_wrong_os", selected: wrongOS, unselected: correct, wantError: "platform windows/arm64 does not match required platform linux/arm64"},
		{name: "selected_empty_platform", selected: emptyPlatform, unselected: correct, wantError: "platform / does not match required platform linux/arm64"},
		{name: "selected_wrong_digest", selected: wrongDigest, unselected: correct, wantError: "do not contain exact reference"},
		{name: "selected_missing_image", selected: ociInspectFixtureResponse{Error: "Error response from daemon: No such image"}, unselected: correct, wantError: "No such image"},
		{name: "selected_platform_unavailable", selected: ociInspectFixtureResponse{Error: "image does not provide the specified platform (linux/arm64)"}, unselected: correct, wantError: "does not provide the specified platform"},
		{name: "selected_daemon_unreachable", selected: ociInspectFixtureResponse{Error: "Cannot connect to the Docker daemon"}, unselected: correct, wantError: "Cannot connect"},
		{name: "client_fallback_passes", selected: unknownFlag, unselected: correct, fallback: true},
		{name: "client_fallback_platform_mismatch", selected: unknownFlag, unselected: wrongPlatform, fallback: true, wantError: "does not match required platform"},
		{name: "daemon_fallback_passes", selected: oldAPI, unselected: correct, fallback: true},
		{name: "daemon_fallback_platform_mismatch", selected: oldAPI, unselected: wrongPlatform, fallback: true, wantError: "does not match required platform"},
		{name: "daemon_fallback_empty_platform", selected: oldAPI, unselected: emptyPlatform, fallback: true, wantError: "platform / does not match required platform"},
		{name: "daemon_fallback_wrong_digest", selected: oldAPI, unselected: wrongDigest, fallback: true, wantError: "do not contain exact reference"},
		{name: "daemon_fallback_missing_image", selected: oldAPI, unselected: ociInspectFixtureResponse{Error: "No such image"}, fallback: true, wantError: "No such image"},
		{name: "explicit_variant_passes", required: "linux/arm/v7", selected: variant, unselected: variant},
		{name: "explicit_variant_mismatch", required: "linux/arm/v6", selected: variant, unselected: variant, wantError: "platform linux/arm/v7 does not match required platform linux/arm/v6"},
		{name: "explicit_variant_missing", required: "linux/arm64/v8", selected: correct, unselected: correct, wantError: "does not match required platform"},
		{name: "unexpected_variant", required: "linux/arm", selected: variant, unselected: variant, wantError: "does not match required platform"},
		{name: "fallback_explicit_variant_passes", required: "linux/arm/v7", selected: oldAPI, unselected: variant, fallback: true},
		{name: "fallback_explicit_variant_mismatch", required: "linux/arm/v6", selected: oldAPI, unselected: variant, fallback: true, wantError: "does not match required platform"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			required := tc.required
			if required == "" {
				required = "linux/arm64"
			}
			work := t.TempDir()
			path := filepath.Join(work, "inspect.json")
			body, err := json.Marshal(ociInspectFixture{Selected: tc.selected, Unselected: tc.unselected})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, body, 0o600); err != nil {
				t.Fatal(err)
			}
			engine := append(checkerProcessFixtureCommand(t, "oci-inspect"), path)
			err = verifyLocalOCIImage(engine, testRuntimeImage, testRuntimeClosure, required, time.Second, work)
			if tc.wantError == "" && err != nil {
				t.Errorf("verification failed: %v", err)
			} else if tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Errorf("verification error = %v, want %q", err, tc.wantError)
			}
			log, readErr := os.ReadFile(path + ".calls")
			if readErr != nil {
				t.Fatal(readErr)
			}
			dec := json.NewDecoder(strings.NewReader(string(log)))
			var calls [][]string
			for dec.More() {
				var call []string
				if err := dec.Decode(&call); err != nil {
					t.Fatal(err)
				}
				calls = append(calls, call)
			}
			wantCalls := 1
			if tc.fallback {
				wantCalls = 2
			}
			if len(calls) != wantCalls {
				t.Fatalf("inspect calls = %v, want %d", calls, wantCalls)
			}
			for i, call := range calls {
				wantPrefix := []string{"image", "inspect", "--platform", required, "--format"}
				if i == 1 {
					wantPrefix = []string{"image", "inspect", "--format"}
				}
				if len(call) != len(wantPrefix)+2 || !reflect.DeepEqual(call[:len(wantPrefix)], wantPrefix) || call[len(call)-1] != testRuntimeImage {
					t.Errorf("inspect argv = %v, want prefix %v and exact pinned reference", call, wantPrefix)
				}
			}
		})
	}
}
