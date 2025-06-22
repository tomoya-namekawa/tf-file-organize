package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tomoya-namekawa/terraform-file-organize/internal/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print the version information including build details, git commit, and build date.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.GetBuildInfo())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
