package git

import (
	"github.com/spf13/cobra"
)

var GitCmd = &cobra.Command{
	Use:   "git",
	Short: "A collection of git commands to help you work with git.",
	Long: `A collection of git commands to help you work with git.

This command will help you work with git by providing a collection of commands to help you.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Run help command if no subcommand is provided.
		if len(args) == 0 {
			cmd.Help()
			return
		}
	},
}

func init() {
	GitCmd.AddCommand(createprCmd)
}
