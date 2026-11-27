package assess

import (
	"os"
	"path/filepath"

	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
)

// The snapshot code of sections 18.1 and 18.2, shared by machinery assess and
// machinery adopt --rehearse (section 17.3). verb names the command in the
// refusals.

// ResolveRoot finds the project: the -C directory, else the nearest directory
// upward that holds machinery.yaml, else the top of the git work tree.
func ResolveRoot(projectFlag string) (string, error) {
	root := projectFlag
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		root = wd
		if r, ferr := project.Find(wd); ferr == nil {
			root = r
		} else if repo, err := gitx.IsRepo(wd); err != nil {
			return "", err
		} else if repo {
			if root, err = gitx.TopLevel(wd); err != nil {
				return "", err
			}
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if info, serr := os.Stat(abs); serr != nil || !info.IsDir() {
		return "", usagef("%s is not a directory", root)
	}
	return gitx.RealPath(abs), nil
}

// HeadOf returns the short sha of HEAD, refusing a directory without a commit.
func HeadOf(root, verb string) (string, error) {
	head, err := gitx.HeadIfAny(root)
	if err != nil {
		return "", err
	}
	if head == "" {
		return "", usagef("%s is not a git repository with a commit; %s reads its HEAD", root, verb)
	}
	return gitx.Short(head), nil
}

// PrepareOut checks --out, or makes the default one under the system temp
// directory (machinery-<verb>-*), outside the project and the repository
// that holds it.
func PrepareOut(root, out, verb string) (string, error) {
	containers := []string{root}
	repo, err := gitx.IsRepo(root)
	if err != nil {
		return "", err
	}
	if repo {
		top, err := gitx.TopLevel(root)
		if err != nil {
			return "", err
		}
		gd, err := gitx.CommonDir(root)
		if err != nil {
			return "", err
		}
		containers = append(containers, gitx.RealPath(top), gitx.RealPath(gd))
	}
	inside := func(p string) string {
		rp := gitx.RealPath(p)
		for _, c := range containers {
			if gitx.Within(rp, c) {
				return c
			}
		}
		return ""
	}
	if out == "" {
		tmp := os.TempDir()
		if c := inside(tmp); c != "" {
			return "", usagef("the system temp directory %s is inside %s; %s never writes in the project: name a directory outside it with --out", tmp, c, verb)
		}
		prefix := "machinery-assess-"
		if verb != "assess" {
			prefix = "machinery-rehearse-"
		}
		return os.MkdirTemp("", prefix)
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return "", err
	}
	if c := inside(abs); c != "" {
		return "", usagef("--out %s is inside %s; %s never writes in the project: name a directory outside it", out, c, verb)
	}
	if info, serr := os.Lstat(abs); serr == nil {
		entries, rerr := os.ReadDir(abs)
		if !info.IsDir() || rerr != nil || len(entries) > 0 {
			return "", usagef("--out %s is not empty; name an absent or empty directory", out)
		}
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	return abs, nil
}

// TakeSnapshot extracts HEAD into <out>/snapshot, writes cfg there as
// machinery.yaml and adoption as design/adoption.yaml (removes each when it
// is nil) and makes it a repository with one commit. It returns the snapshot
// and the symbolic links left out.
func TakeSnapshot(root, out, head string, cfg, adoption []byte) (string, []string, error) {
	snap := filepath.Join(out, "snapshot")
	if err := os.MkdirAll(snap, 0o755); err != nil {
		return "", nil, err
	}
	_, outside, err := gitx.Archive(root, snap)
	if err != nil {
		return "", nil, err
	}
	for _, f := range []struct {
		rel  string
		data []byte
	}{{project.ConfigName, cfg}, {project.AdoptionName, adoption}} {
		path := filepath.Join(snap, filepath.FromSlash(f.rel))
		if err := os.RemoveAll(path); err != nil {
			return "", nil, err
		}
		if f.data == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", nil, err
		}
		if err := os.WriteFile(path, f.data, 0o644); err != nil {
			return "", nil, err
		}
	}
	if err := gitx.InitCommit(snap, "snapshot of "+head); err != nil {
		return "", nil, err
	}
	return snap, outside, nil
}
