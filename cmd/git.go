package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// gitCmd represents the git command
var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "A collection of git commands to help you work with git.",
	Long: `A collection of git commands to help you work with git.

This command will help you work with git by providing a collection of commands to help you.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("git called")
	},
}

func init() {
	rootCmd.AddCommand(gitCmd)
	gitCmd.AddCommand(createprCmd)
}
