package cmd

import (
	"github.com/spf13/cobra"

	"github.com/tomoya-namekawa/tf-file-organize/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "tf-file-organize",
	Short:   "Organize Terraform files by resource type",
	Version: version.GetVersion(),
	Long: `A CLI tool to split Terraform files into separate files organized by resource type.
Each resource type will be placed in its own file following naming conventions.

Available commands:
  run             Organize Terraform files
  plan            Show what would be done without actually creating files
  validate-config Validate configuration file
  version         Show version information

Use "tf-file-organize <command> --help" for more information about a command.`,
}

// Execute runs the root command and handles CLI argument parsing.
func Execute() error {
	// Enable version flag
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	return rootCmd.Execute()
}
