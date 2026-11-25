package formal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/version"
)

// Tool is a pinned formal tool (section 16.8).
type Tool struct {
	Name    string
	Version string
	File    string // relative to the tools directory
	SHA256  string
	URL     string
}

// The pinned tools.
var (
	TLATool   = Tool{"tla2tools", version.TLAVersion, "jars/tla2tools.jar", version.TLASHA256, "https://github.com/tlaplus/tlaplus/releases/download/" + version.TLAVersion + "/tla2tools.jar"}
	AlloyTool = Tool{"alloy", version.AlloyVer, "jars/alloy.jar", version.AlloySHA256, "https://github.com/AlloyTools/org.alloytools.alloy/releases/download/" + version.AlloyVer + "/org.alloytools.alloy.dist.jar"}
	Tools     = []Tool{TLATool, AlloyTool}
)

// ToolFor returns the pinned tool of a spec's tool name.
func ToolFor(name string) Tool {
	if name == "alloy" {
		return AlloyTool
	}
	return TLATool
}

// Label is how a result names the tool.
func (t Tool) Label() string { return t.Name + " " + t.Version }

// Path is the absolute path of the tool's file.
func (t Tool) Path(toolsDir string) string {
	return filepath.Join(toolsDir, filepath.FromSlash(t.File))
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Verify checks that the tool's file exists and has the pinned checksum.
func (t Tool) Verify(toolsDir string) error {
	got, err := fileSHA(t.Path(toolsDir))
	if err != nil {
		return fmt.Errorf("%s: missing (run machinery formal tools)", t.File)
	}
	if got != t.SHA256 {
		return fmt.Errorf("%s: checksum mismatch", t.File)
	}
	return nil
}

// Fetch downloads the tool into the tools directory and verifies it. The
// environment variable MACHINERY_TOOLS_MIRROR names a base URL (https or
// file) that holds the files under their base names.
func (t Tool) Fetch(toolsDir string) error {
	src := t.URL
	if mirror := os.Getenv("MACHINERY_TOOLS_MIRROR"); mirror != "" {
		src = strings.TrimSuffix(mirror, "/") + "/" + filepath.Base(t.File)
	}
	dest := t.Path(toolsDir)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".fetch-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	var body io.ReadCloser
	if strings.HasPrefix(src, "file://") {
		body, err = os.Open(strings.TrimPrefix(src, "file://"))
		if err != nil {
			tmp.Close()
			return fmt.Errorf("%s: cannot read %s", t.File, src)
		}
	} else {
		client := &http.Client{Timeout: 5 * time.Minute}
		resp, herr := client.Get(src)
		if herr != nil {
			tmp.Close()
			return fmt.Errorf("%s: download failed: %v", t.File, herr)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			tmp.Close()
			return fmt.Errorf("%s: download failed: %s", t.File, resp.Status)
		}
		body = resp.Body
	}
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(tmp, h), body)
	body.Close()
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("%s: download failed: %v", t.File, err)
	}
	if hex.EncodeToString(h.Sum(nil)) != t.SHA256 {
		return fmt.Errorf("%s: checksum mismatch", t.File)
	}
	return os.Rename(tmp.Name(), dest)
}

// ToolError says why a formal run could not be demonstrated.
type ToolError struct{ Reason string }

func (e *ToolError) Error() string { return e.Reason }

// Ready checks the Java runtime and the jar of one tool.
func Ready(p *project.Project, t Tool) error {
	if err := t.Verify(p.ToolsDir()); err != nil {
		return &ToolError{err.Error()}
	}
	if out, err := exec.Command(p.Cfg.Tools.Java, "-version").CombinedOutput(); err != nil {
		_ = out
		return &ToolError{fmt.Sprintf("java runtime not found (%s)", p.Cfg.Tools.Java)}
	}
	return nil
}
