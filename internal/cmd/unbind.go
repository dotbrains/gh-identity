package cmd

import (
	"fmt"

	"github.com/dotbrains/gh-identity/internal/config"
	"github.com/dotbrains/gh-identity/internal/gitconfig"
	"github.com/spf13/cobra"
)

func newUnbindCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unbind [<path>]",
		Short: "Remove the binding for a directory",
		Long:  "Remove the binding for a directory (defaults to $PWD).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dirPath := "."
			if len(args) == 1 {
				dirPath = args[0]
			}
			return runUnbind(dirPath)
		},
	}
}

func runUnbind(dirPath string) error {
	expanded, err := config.ExpandPath(dirPath)
	if err != nil {
		return err
	}

	bindings, err := config.LoadBindings()
	if err != nil {
		return err
	}

	if err := bindings.RemoveBinding(expanded); err != nil {
		return err
	}
	if err := bindings.Save(); err != nil {
		return err
	}

	// Remove includeIf from global gitconfig.
	gcPath, err := gitconfig.GlobalGitconfigPath()
	if err == nil {
		_ = gitconfig.RemoveIncludeIf(gcPath, expanded)
	}

	fmt.Printf("✅ Unbound %s\n", expanded)
	return nil
}
