package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/RamXX/machinery/internal/install"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		versionFlag string
		repo        string
		installDir  string
		homes       []string
		targets     []string
		copyAll     bool
		skipPlugins bool
	)
	c := &cobra.Command{
		Use:   "update",
		Short: "Force-refresh machinery and every installed agent harness",
		Long: `Download the requested machinery release, verify its published SHA-256
checksum and reported version, atomically replace this binary, then use the new
binary to refresh every recorded direct skill, role, command, and host adapter
from the same release.

With no selectors, update uses the installation receipt plus standard-path
discovery. Explicit --home and --target flags restrict the harness refresh to
those placements; unlike install, both may be supplied together. The update is
forced even when the requested version is already installed. Host-owned Claude
Code and Codex plugin caches are refreshed through their CLIs when detected,
never overwritten directly.

  machinery update
  machinery update --version v0.3.4
  machinery update --target all
  machinery update --home ~/.agents --target codex`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if copyAll && len(homes) == 0 && len(targets) == 0 {
				return fmt.Errorf("--copy requires an explicit --home or --target selection; recorded installs keep their recorded topology")
			}
			destination := ""
			if installDir != "" {
				name := "machinery"
				if runtime.GOOS == "windows" {
					name += ".exe"
				}
				destination = filepath.Join(installDir, name)
			}
			_, err := install.Update(install.UpdateOptions{
				Version:     versionFlag,
				Repo:        repo,
				Executable:  destination,
				Homes:       homes,
				Targets:     targets,
				Copy:        copyAll,
				SkipPlugins: skipPlugins,
				Out:         cmd.OutOrStdout(),
			})
			return err
		},
	}
	c.Flags().StringVar(&versionFlag, "version", "latest", "release tag to install, or latest")
	c.Flags().StringVar(&repo, "repo", "", "source repo owner/name (default RamXX/machinery)")
	c.Flags().StringVar(&installDir, "install-dir", "", "binary destination directory (default: replace the running executable)")
	c.Flags().StringArrayVar(&homes, "home", nil, "direct agent home to refresh (repeatable; explicit selectors restrict auto-discovery)")
	c.Flags().StringArrayVar(&targets, "target", nil, "native host target to refresh: claude, codex, opencode, or all (repeatable)")
	c.Flags().BoolVar(&copyAll, "copy", false, "with explicit selectors, copy rather than symlink secondary homes")
	c.Flags().BoolVar(&skipPlugins, "skip-plugins", false, "do not ask detected Claude Code or Codex plugin managers to refresh machinery")
	return c
}
