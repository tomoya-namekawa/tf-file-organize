package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tomoya-namekawa/tf-file-organize/internal/config"
	"github.com/tomoya-namekawa/tf-file-organize/internal/validation"
)

// validateConfigCmd represents the validate-config command
var validateConfigCmd = &cobra.Command{
	Use:   "validate-config <config-file>",
	Short: "Validate configuration file",
	Long: `Validate the syntax and content of a tf-file-organize configuration file.

This command checks:
- YAML syntax
- Required fields
- Pattern validity
- Group name uniqueness
- Filename conflicts
- Exclude file pattern validity

If the configuration is valid, a summary of the configuration will be displayed.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		configFile := args[0]
		if err := runValidateConfig(configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Configuration validation failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(validateConfigCmd)
}

func runValidateConfig(configPath string) error {
	// Basic path validation
	if err := validation.ValidateConfigPath(configPath); err != nil {
		return err
	}

	fmt.Printf("Validating configuration file: %s\n", configPath)

	// Load and validate configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Perform additional validation
	if err := config.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Display configuration summary
	printConfigSummary(cfg)

	fmt.Println("✅ Configuration is valid!")
	return nil
}

func printConfigSummary(cfg *config.Config) {
	fmt.Println("\n📋 Configuration Summary:")
	fmt.Printf("  Groups: %d\n", len(cfg.Groups))
	fmt.Printf("  Exclude File Patterns: %d\n", len(cfg.ExcludeFiles))

	if len(cfg.Groups) > 0 {
		fmt.Println("\n📁 Groups:")
		for i, group := range cfg.Groups {
			fmt.Printf("  %d. %s → %s\n", i+1, group.Name, group.Filename)
			for _, pattern := range group.Patterns {
				fmt.Printf("     - %s\n", pattern)
			}
		}
	}

	if len(cfg.ExcludeFiles) > 0 {
		fmt.Println("\n🚫 Exclude File Patterns:")
		for i, pattern := range cfg.ExcludeFiles {
			fmt.Printf("  %d. %s\n", i+1, pattern)
		}
	}
}
