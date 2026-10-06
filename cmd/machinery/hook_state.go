package main

import (
	"github.com/RamXX/machinery/internal/hook"
	"github.com/spf13/cobra"
)

func newHookStateCmd() *cobra.Command {
	command := &cobra.Command{Use: "hook-state", Short: "Inspect and transition the durable governance hook store", Args: cobra.NoArgs}
	var root, from string
	var rebind bool
	adopt := &cobra.Command{Use: "adopt", Short: "Reaffirm the recorded store binding while preserving obligations", Args: cobra.NoArgs}
	adopt.Flags().StringVar(&root, "root", "", "project root whose retained obligations are reported")
	adopt.Flags().StringVar(&from, "from", "", "restore the recorded store from a quarantine directory")
	adopt.Flags().BoolVar(&rebind, "rebind-identity", false,
		"accept a verified store whose native directory identity changed (moved or restored to another filesystem); its generation must match the independent initialization marker")
	_ = adopt.MarkFlagRequired("root")
	adopt.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		return hook.AdoptStateWith(output.stdout, root, from, hook.AdoptOptions{RebindIdentity: rebind})
	}
	command.AddCommand(adopt)

	var releaseRoot string
	var tokens []string
	var orphaned bool
	release := &cobra.Command{
		Use:   "release",
		Short: "Release in-flight tool tokens no session can complete; the gate obligation stays armed",
		Long: "Release removes in-flight tool tokens that no agent session can complete any more (a host killed\n" +
			"mid tool call, a plugin disabled between PreToolUse and PostToolUse, a ledger written by an older\n" +
			"release). The project's design/impl gate obligation stays armed, so the next Stop still runs the\n" +
			"gates. Every release is journaled with time, operator, host, and the released tokens.\n" +
			"Use --orphaned only when no agent session in the project is running a tool.",
		Args: cobra.NoArgs,
	}
	release.Flags().StringVar(&releaseRoot, "root", "", "project root whose tokens are released")
	release.Flags().StringArrayVar(&tokens, "token", nil, "release one token, named in full or by a unique prefix of at least 12 hex characters (repeatable)")
	release.Flags().BoolVar(&orphaned, "orphaned", false, "release every in-flight token recorded for the project")
	_ = release.MarkFlagRequired("root")
	release.MarkFlagsMutuallyExclusive("token", "orphaned")
	release.MarkFlagsOneRequired("token", "orphaned")
	release.RunE = func(cmd *cobra.Command, args []string) (retErr error) {
		output := trackCommandOutput()
		defer func() { retErr = output.join(retErr) }()
		return hook.ReleaseState(output.stdout, releaseRoot, tokens, orphaned)
	}
	command.AddCommand(release)
	return command
}
