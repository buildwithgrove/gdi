// ---------------------------------------------------------------------------
// File: root.go
// Package: cmd
//
// Overview:
//
// The 🌿 Grove Developer Interface (GDI) 🌿 is a command-line tool designed to streamline
// internal developer workflows at Grove. GDI is intended to help developers quickly perform
// routine operations and maintain consistency across projects.
//
// DEV_NOTE:
//
// This repo is also intended to be a living project and should be updated to incorporate
// any of our own scripts, hacks, time-saving features, etc. that we use in our local
// development workflows and that could benefit the entire team to share in this CLI
//
// ---------------------------------------------------------------------------
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/buildwithgrove/gdi/cmd/config"
	"github.com/buildwithgrove/gdi/cmd/git"
)

var rootCmd = &cobra.Command{
	Use:     "gdi",
	Short:   "Grove Developer Interface - streamline your development workflows",
	Long:    "", //Assigned in generateLongDescription()
	Version: "0.0.1",
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Toggle verbose mode or other options")
	rootCmd.AddCommand(git.GitCmd)
	rootCmd.AddCommand(config.ConfigCmd)
	rootCmd.Long = generateLongDescription()

	if !config.ConfigExists() {
		config.RunFirstTimeSetup()
		return
	}
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// generateLongDescription generates the long description for the root command.
func generateLongDescription() string {
	var sb strings.Builder
	sb.WriteString(`Grove Developer Interface (GDI) is a comprehensive CLI tool designed for internal
	development at Grove. It provides users with a unified approach to manage configuration
	settings, execute Git operations, and interact with integrated LLM providers for tasks
	such as automated pull request generation.

	Available Commands:
	`)
	appendCommands(&sb, rootCmd, "")
	return sb.String()
}

// appendCommands appends the commands to the long description.
func appendCommands(sb *strings.Builder, cmd *cobra.Command, prefix string) {
	for _, c := range cmd.Commands() {
		if !c.Hidden {
			sb.WriteString(fmt.Sprintf("%s  %-10s %s\n", prefix, c.Name(), c.Short))
			appendCommands(sb, c, prefix+"  ")
		}
	}
}
