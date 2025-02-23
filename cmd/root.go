/*
Copyright © 2025 Pascal van Leeuwen <pascal@grove.city>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/buildwithgrove/gdi/cmd/git"
)

var rootCmd = &cobra.Command{
	Use:   "gdi",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Version: "0.0.1",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.AddCommand(git.GitCmd)
}
