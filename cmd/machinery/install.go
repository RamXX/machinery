package main

import (
	"fmt"
	"strings"

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
version (a -dev binary uses the latest release).

  machinery install
  machinery install --target codex
  machinery install --target opencode
  machinery install --target all
  machinery install --home ~/.claude
  machinery install --from . --copy        # from a local checkout, real copies everywhere`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			v := verFlag
			if v == "" {
				v = version // this binary's version (main.version)
			}
			return install.Install(install.Options{
				Homes:   homes,
				Targets: targets,
				From:    from,
				Copy:    copyAll,
				Version: v,
				Repo:    repo,
				Out:     cmd.OutOrStdout(),
				Record:  true,
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(targets) > 0 {
				if len(homes) > 0 {
					return fmt.Errorf("--home and --target cannot be combined")
				}
				if err := install.UninstallTargets(targets, cmd.OutOrStdout()); err != nil {
					return err
				}
				if err := install.ForgetTargetInstalls(targets); err != nil {
					return err
				}
				// Target removal also owns these direct paths: Claude always
				// removes ~/.claude, while a complete selection removes the
				// shared ~/.agents copy too.
				selected := map[string]bool{}
				for _, target := range targets {
					selected[strings.ToLower(strings.TrimSpace(target))] = true
				}
				all := selected["all"] || (selected["claude"] && selected["codex"] && selected["opencode"])
				removeHomes := []string{}
				defaults := install.DefaultHomes()
				if all {
					removeHomes = append(removeHomes, defaults...)
				} else if selected["claude"] && len(defaults) > 1 {
					removeHomes = append(removeHomes, defaults[1])
				}
				if len(removeHomes) > 0 {
					return install.ForgetHomeInstalls(removeHomes)
				}
				return nil
			}
			if err := install.Uninstall(homes, cmd.OutOrStdout()); err != nil {
				return err
			}
			return install.ForgetHomeInstalls(homes)
		},
	}
	c.Flags().StringArrayVar(&homes, "home", nil, "agent home to remove from (repeatable). Default: ~/.agents ~/.claude")
	c.Flags().StringArrayVar(&targets, "target", nil, "host-aware removal: claude, codex, opencode, or all (repeatable; cannot combine with --home)")
	return c
}
