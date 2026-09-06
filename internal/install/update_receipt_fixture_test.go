package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func updatePlacementOptions(args []string) (Options, error) {
	opts := Options{Record: true}
	if len(args) == 0 || args[0] != "install" {
		return opts, fmt.Errorf("expected install: %q", args)
	}
	for i := 1; i < len(args); i++ {
		key := args[i]
		if key == "--copy" {
			if opts.Copy {
				return opts, fmt.Errorf("duplicate --copy")
			}
			opts.Copy = true
			continue
		}
		if key != "--from" && key != "--home" && key != "--target" {
			return opts, fmt.Errorf("unexpected install argument %q", key)
		}
		i++
		if i == len(args) || args[i] == "" || strings.HasPrefix(args[i], "--") {
			return opts, fmt.Errorf("missing value for %s", key)
		}
		switch key {
		case "--from":
			if opts.From != "" || !filepath.IsAbs(args[i]) {
				return opts, fmt.Errorf("invalid or repeated --from")
			}
			opts.From = args[i]
		case "--home":
			if !filepath.IsAbs(args[i]) {
				return opts, fmt.Errorf("nonabsolute --home")
			}
			opts.Homes = append(opts.Homes, args[i])
		case "--target":
			opts.Targets = append(opts.Targets, args[i])
		}
	}
	if opts.From == "" || (len(opts.Homes) == 0) == (len(opts.Targets) == 0) {
		return opts, fmt.Errorf("expected source and exactly one selector kind: %q", args)
	}
	if _, err := parseTargetsOptional(opts.Targets); err != nil {
		return opts, err
	}
	return opts, nil
}

// Component observer only: retain the parent's active transaction and use its
// real post-image-aware placement operations, without reentering Install.
func runUpdatePlacement(opts Options) (retErr error) {
	source, err := resolveInstallSource(opts, io.Discard)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, source.cleanup()) }()
	if len(opts.Targets) > 0 {
		if err := installTargets(opts.Targets, source.path, opts.Copy, io.Discard, nil); err != nil {
			return err
		}
	} else {
		homes, err := absHomes(opts.Homes)
		if err != nil {
			return err
		}
		for i, home := range homes {
			if i == 0 || opts.Copy {
				err = placeReal(home, source.path, io.Discard)
			} else {
				err = placeLinks(home, homes[0], io.Discard)
			}
			if err != nil {
				return err
			}
		}
	}
	return source.verify()
}

// Invoked only by the downloaded shell candidate in the execution-wiring test.
// An ordinary uninvoked test run is scaffolding, not behavioral proof.
func TestUpdatePlacementChildHelper(t *testing.T) {
	i := slices.Index(os.Args, "--")
	if i < 0 {
		return
	}
	opts, err := updatePlacementOptions(os.Args[i+1:])
	if err == nil {
		err = EnsureActivationConsistency()
	}
	if err == nil {
		err = Install(opts)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("actual placement child completed")
}

// Download the complete native fixture, not an unrelated local placement source.
func updateNativeReleaseServer(t *testing.T, tag string, candidate []byte, badChecksum bool) *httptest.Server {
	t.Helper()
	source := fakeSource(t)
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = "machinery/" + filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, file)
		return errors.Join(copyErr, file.Close())
	})
	if err := errors.Join(err, tw.Close(), gz.Close()); err != nil {
		t.Fatal(err)
	}
	asset, err := releaseAssetName()
	if err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(candidate))
	if badChecksum {
		sum = strings.Repeat("0", 64)
	}
	tarball := archive.Bytes()
	sourceSum := fmt.Sprintf("%x", sha256.Sum256(tarball))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/tags/"+tag):
			_, _ = fmt.Fprintf(w, `{"tag_name":%q}`, tag)
		case strings.HasSuffix(r.URL.Path, "/releases/download/"+tag+"/"+asset):
			_, _ = w.Write(candidate)
		case strings.HasSuffix(r.URL.Path, "/releases/download/"+tag+"/checksums-sha256.txt"):
			_, _ = fmt.Fprintf(w, "%s  %s\n%s  machinery-source.tar.gz\n", sum, asset, sourceSum)
		case strings.HasSuffix(r.URL.Path, "/releases/download/"+tag+"/machinery-source.tar.gz"):
			_, _ = w.Write(tarball)
		default:
			http.NotFound(w, r)
		}
	}))
	oldAPI := apiBase
	apiBase = server.URL
	t.Cleanup(func() { apiBase = oldAPI })
	return server
}

func assertUpdateHomePlacement(t *testing.T, home, canonical string, linked bool) {
	t.Helper()
	for _, rel := range append([]string{"skills/machinery"}, "agents/machinery-fsm-author.md", "agents/machinery-build-writer.md") {
		path := filepath.Join(home, rel)
		if isSymlink(t, path) != linked {
			t.Errorf("placement link mode for %s, want linked=%v", path, linked)
		}
		if linked {
			got, err := os.Readlink(path)
			if err != nil || !sameInstallPath(got, filepath.Join(canonical, rel)) {
				t.Errorf("link destination %s = %s, %v", path, got, err)
			}
		}
	}
	for rel, want := range map[string]string{
		"skills/machinery/SKILL.md":        "---\nname: machinery\n---\n",
		"skills/machinery/references/x.md": "ref\n",
		"agents/machinery-fsm-author.md":   "---\nname: role\ndescription: role\ntools: Read, Write\nmodel: opus\n---\n\ncanonical role body for machinery-fsm-author.md\n",
		"agents/machinery-build-writer.md": "---\nname: role\ndescription: role\ntools: Read, Write\nmodel: opus\n---\n\ncanonical role body for machinery-build-writer.md\n",
	} {
		if got, err := os.ReadFile(filepath.Join(home, rel)); err != nil || string(got) != want {
			t.Errorf("current placement %s/%s = %q, %v", home, rel, got, err)
		}
	}
}

func assertUpdateNativePlacement(t *testing.T, home string) {
	t.Helper()
	for _, role := range RoleDocs {
		for _, path := range []string{filepath.Join(home, ".codex", "agents", strings.TrimSuffix(role, ".md")+".toml"), filepath.Join(home, ".config", "opencode", "agents", role)} {
			got, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(got), "canonical role body for "+role) || isSymlink(t, path) {
				t.Errorf("native rendered role %s = %q, %v", path, got, err)
			}
		}
	}
	for _, command := range openCodeCommands {
		path := filepath.Join(home, ".config", "opencode", "commands", command)
		if got, err := os.ReadFile(path); err != nil || string(got) != "---\ndescription: command\n---\n\ncommand "+command+"\n" {
			t.Errorf("native command %s = %q, %v", path, got, err)
		}
	}
	path := filepath.Join(home, ".config", "opencode", "plugins", "machinery.js")
	if got, err := os.ReadFile(path); err != nil || string(got) != "export const MachineryPlugin = async () => ({})\n" {
		t.Errorf("native plugin = %q, %v", got, err)
	}
}
