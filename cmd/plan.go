package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	planInputFile  string
	planOutputDir  string
	planConfigFile string
	planRecursive  bool
)

// planCmd represents the plan command
var planCmd = &cobra.Command{
	Use:   "plan <input-path>",
	Short: "Show what would be done without actually creating files",
	Long: `Preview the organization of Terraform files without making any changes.

This is equivalent to 'run --dry-run' but as a dedicated subcommand.
Shows which files would be created and how blocks would be organized.

Input can be either a single .tf file or a directory containing .tf files.
By default, only files in the specified directory are processed. Use -r for recursive processing.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		planInputFile = args[0]
		if err := runPlan(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(planCmd)

	// Setup flags for plan command (same as run but without backup since it's dry-run)
	planCmd.Flags().StringVarP(&planOutputDir, "output-dir", "o", "", "Output directory for split files (default: same as input path)")
	planCmd.Flags().StringVarP(&planConfigFile, "config", "c", "", "Configuration file for custom grouping rules")
	planCmd.Flags().BoolVarP(&planRecursive, "recursive", "r", false, "Process directories recursively")
}

func runPlan() error {
	return executeOrganizeFiles(planInputFile, planOutputDir, planConfigFile, planRecursive, true, false)
}
