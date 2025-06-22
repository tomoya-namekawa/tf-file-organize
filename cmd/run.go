package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	runInputFile  string
	runOutputDir  string
	runConfigFile string
	runRecursive  bool
	runBackup     bool
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run <input-path>",
	Short: "Organize Terraform files by resource type",
	Long: `A CLI tool to split Terraform files into separate files organized by resource type.
Each resource type will be placed in its own file following naming conventions.

Input can be either a single .tf file or a directory containing .tf files.
By default, only files in the specified directory are processed. Use -r for recursive processing.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runInputFile = args[0]
		if err := runOrganize(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Setup flags for run command
	runCmd.Flags().StringVarP(&runOutputDir, "output-dir", "o", "", "Output directory for split files (default: same as input path)")
	runCmd.Flags().StringVarP(&runConfigFile, "config", "c", "", "Configuration file for custom grouping rules")
	runCmd.Flags().BoolVarP(&runRecursive, "recursive", "r", false, "Process directories recursively")
	runCmd.Flags().BoolVar(&runBackup, "backup", false, "Backup original files to 'backup' subdirectory before organizing")
}

func runOrganize() error {
	return executeOrganizeFiles(runInputFile, runOutputDir, runConfigFile, runRecursive, false, runBackup)
}
