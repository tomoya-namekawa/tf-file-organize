package cmd

import (
	"fmt"

	"github.com/tomoya-namekawa/tf-file-organize/internal/usecase"
	"github.com/tomoya-namekawa/tf-file-organize/internal/validation"
)

// executeOrganizeFiles validates inputs and executes the organize files usecase
func executeOrganizeFiles(inputPath, outputDir, configFile string, recursive, dryRun, backup bool) error {
	// Validate all inputs first
	if err := validation.ValidateInputPath(inputPath); err != nil {
		return fmt.Errorf("invalid input path: %w", err)
	}

	if err := validation.ValidateOutputPath(outputDir); err != nil {
		return err
	}

	if err := validation.ValidateConfigPath(configFile); err != nil {
		return err
	}

	// Validate flag combinations
	if err := validation.ValidateFlagCombination(outputDir, recursive); err != nil {
		return err
	}

	// Create usecase request
	req := &usecase.OrganizeFilesRequest{
		InputPath:  inputPath,
		OutputDir:  outputDir,
		ConfigFile: configFile,
		DryRun:     dryRun,
		Recursive:  recursive,
		Backup:     backup,
	}

	// Execute usecase
	uc := usecase.NewOrganizeFilesUsecase()
	_, err := uc.Execute(req)
	if err != nil {
		return err
	}

	return nil
}
