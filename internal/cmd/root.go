// Package cmd provides the cobra command tree for gh-identity.
package cmd

import (
	"github.com/dotbrains/gh-identity/internal/ghauth"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for gh identity.
func NewRootCmd() *cobra.Command {
	auth := ghauth.NewGHAuth()

	root := &cobra.Command{
		Use:   "identity",
		Short: "Manage multiple GitHub identities",
		Long:  `gh-identity provides seamless multi-account management, automatic context-based account switching, and per-directory identity binding.`,
	}

	root.AddCommand(
		newInitCmd(auth),
		newProfileCmd(auth),
		newBindCmd(),
		newUnbindCmd(),
		newSwitchCmd(auth),
		newStatusCmd(auth),
		newCloneCmd(auth),
		newDoctorCmd(auth),
	)

	return root
}
