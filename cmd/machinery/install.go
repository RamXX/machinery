package main

import (
	"fmt"

	"github.com/RamXX/machinery/internal/install"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		homes   []string
		targets []string
		from    string
		copyAll bool
		verFlag string
		repo    string
	)
	c := &cobra.Command{
		Use:   "install",
		Short: "Install the machinery skill + role docs into your agent home(s)",
		Long: `Install the machinery skill and the two role docs into your agent home(s).

With no --target, the first home holds the real files and the rest are
symlinked to it, preserving the original ~/.agents + ~/.claude behavior. Use
--target to install the host-specific assets machinery supports for that host.
With no --from, files are fetched from the release that matches this binary's
version, falling back to the latest release when that version has no
published release yet; an explicit --version is fetched exactly or fails.

  machinery install
  machinery install --target codex
  machinery install --target opencode
  machinery install --target all
  machinery install --home ~/.claude
  machinery install --from . --copy        # from a local checkout, real copies everywhere`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (retErr error) {
			output := trackOutput(cmd.OutOrStdout(), cmd.ErrOrStderr())
			defer func() { retErr = output.join(retErr) }()
			v := verFlag
			if v == "" {
				v = version // this binary's version (main.version)
			}
			return install.Install(install.Options{
				Homes:           homes,
				Targets:         targets,
				From:            from,
				Copy:            copyAll,
				Version:         v,
				VersionExplicit: verFlag != "",
				Repo:            repo,
				Out:             output.stdout,
				Record:          true,
			})
		},
	}
	c.Flags().StringArrayVar(&homes, "home", nil, "agent home to install into (repeatable; first is canonical). Default: ~/.agents ~/.claude")
	c.Flags().StringArrayVar(&targets, "target", nil, "host-aware installation: claude, codex, opencode, or all (repeatable; cannot combine with --home)")
	c.Flags().StringVar(&from, "from", "", "install from a local checkout dir (with skills/ and agents/) instead of downloading")
	c.Flags().BoolVar(&copyAll, "copy", false, "copy into every home instead of symlinking the non-canonical ones")
	c.Flags().StringVar(&verFlag, "version", "", "release tag to fetch (default: this binary's version, else latest)")
	c.Flags().StringVar(&repo, "repo", "", "source repo owner/name (default RamXX/machinery)")
	return c
}

func newUninstallCmd() *cobra.Command {
	var (
		homes   []string
		targets []string
	)
	c := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the machinery skill + role docs from your agent home(s)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) (retErr error) {
			output := trackOutput(cmd.OutOrStdout(), cmd.ErrOrStderr())
			defer func() { retErr = output.join(retErr) }()
			if len(targets) > 0 {
				if len(homes) > 0 {
					return fmt.Errorf("--home and --target cannot be combined")
				}
				return install.UninstallTargets(targets, output.stdout)
			}
			return install.Uninstall(homes, output.stdout)
		},
	}
	c.Flags().StringArrayVar(&homes, "home", nil, "agent home to remove from (repeatable). Default: ~/.agents ~/.claude")
	c.Flags().StringArrayVar(&targets, "target", nil, "host-aware removal: claude, codex, opencode, or all (repeatable; cannot combine with --home)")
	return c
}
