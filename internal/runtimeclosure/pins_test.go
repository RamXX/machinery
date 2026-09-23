package runtimeclosure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJavaPinSpellingsAgree holds the Temurin pin to one version: the
// numeric major/feature/security constants, the archive table and the
// repository's .java-runtime-pin file must all spell the same build.
func TestJavaPinSpellingsAgree(t *testing.T) {
	if want := fmt.Sprintf("%d.%d.%d.", RequiredJavaMajor, RequiredJavaFeature, RequiredJavaSecurity); !strings.HasPrefix(pinnedJavaVersion, want) {
		t.Fatalf("pinnedJavaVersion %s does not start with the numeric pin %s", pinnedJavaVersion, want)
	}
	assetVersion := strings.ReplaceAll(PinnedJavaRuntimeVersion, "+", "_")
	assetPrefix := fmt.Sprintf("OpenJDK%dU-jdk_", RequiredJavaMajor)
	for platform, pin := range javaArchivePins {
		if !strings.HasPrefix(pin.asset, assetPrefix) || !strings.Contains(pin.asset, "_hotspot_"+assetVersion+".") {
			t.Errorf("%s archive %s is not the pinned Temurin %s asset", platform, pin.asset, PinnedJavaRuntimeVersion)
		}
		if len(pin.sha) != 64 {
			t.Errorf("%s archive sha256 %q is not a digest", platform, pin.sha)
		}
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", ".java-runtime-pin"))
	if err != nil {
		t.Fatal(err)
	}
	file := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf(".java-runtime-pin line %q is not KEY=VALUE", line)
		}
		file[key] = value
	}
	if file["JAVA_RUNTIME_VERSION"] != PinnedJavaRuntimeVersion {
		t.Errorf(".java-runtime-pin version %q, want %q", file["JAVA_RUNTIME_VERSION"], PinnedJavaRuntimeVersion)
	}
	if file["JAVA_RUNTIME_RELEASE_TAG"] != pinnedJavaReleaseTag {
		t.Errorf(".java-runtime-pin release tag %q, want %q", file["JAVA_RUNTIME_RELEASE_TAG"], pinnedJavaReleaseTag)
	}
	for platform, pin := range javaArchivePins {
		key := "JAVA_RUNTIME_" + strings.ToUpper(strings.ReplaceAll(platform, "/", "_")) + "_SHA256"
		if file[key] != pin.sha {
			t.Errorf(".java-runtime-pin %s = %q, want the archive table's %q", key, file[key], pin.sha)
		}
	}
}

// TestDerivedRuntimeIdentitiesAgree holds every composite identity to the
// single pins it is built from.
func TestDerivedRuntimeIdentitiesAgree(t *testing.T) {
	if !strings.HasPrefix(RequiredOTPVersion, RequiredOTPMajor+".") {
		t.Errorf("OTP pin %s is not in the pinned release series %s", RequiredOTPVersion, RequiredOTPMajor)
	}
	if want := RequiredElixirVersion + "/" + RequiredOTPVersion + "/" + RequiredErtsVersion; ElixirIdentityVersion != want {
		t.Errorf("Elixir identity %s, want %s", ElixirIdentityVersion, want)
	}
	if RequiredNodeVersion != "v"+RequiredNodeRelease {
		t.Errorf("node --version spelling %s does not match release %s", RequiredNodeVersion, RequiredNodeRelease)
	}
	if want := RequiredNodeRelease + "/" + RequiredTypeScriptVersion; TypeScriptIdentityVersion != want {
		t.Errorf("TypeScript identity %s, want %s", TypeScriptIdentityVersion, want)
	}
}
