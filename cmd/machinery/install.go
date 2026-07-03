package main

import (
	"github.com/RamXX/machinery/internal/install"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		homes   []string
		from    string
		copyAll bool
		verFlag string
		repo    string
	)
	c := &cobra.Command{
		Use:   "install",
		Short: "Install the machinery skill + role docs into your agent home(s)",
		Long: `Install the machinery skill and the two role docs into your agent home(s).

The first home holds the real files and the rest are symlinked to it, so there
is one copy to update. With no --from, the files are fetched from the release
that matches this binary's version (a -dev binary uses the latest release).

  machinery install
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
				From:    from,
				Copy:    copyAll,
				Version: v,
				Repo:    repo,
				Out:     cmd.OutOrStdout(),
			})
		},
	}
	c.Flags().StringArrayVar(&homes, "home", nil, "agent home to install into (repeatable; first is canonical). Default: ~/.agents ~/.claude")
	c.Flags().StringVar(&from, "from", "", "install from a local checkout dir (with skills/ and agents/) instead of downloading")
	c.Flags().BoolVar(&copyAll, "copy", false, "copy into every home instead of symlinking the non-canonical ones")
	c.Flags().StringVar(&verFlag, "version", "", "release tag to fetch (default: this binary's version, else latest)")
	c.Flags().StringVar(&repo, "repo", "", "source repo owner/name (default RamXX/machinery)")
	return c
}

func newUninstallCmd() *cobra.Command {
	var homes []string
	c := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the machinery skill + role docs from your agent home(s)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return install.Uninstall(homes, cmd.OutOrStdout())
		},
	}
	c.Flags().StringArrayVar(&homes, "home", nil, "agent home to remove from (repeatable). Default: ~/.agents ~/.claude")
	return c
}
