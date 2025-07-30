package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"autoteam/internal/config"
)

//go:embed templates/*
var templateFS embed.FS

type Generator struct {
	fileOps *FileOperations
}

func New() *Generator {
	return &Generator{
		fileOps: NewFileOperations(),
	}
}

func (g *Generator) GenerateCompose(cfg *config.Config) error {
	// Ensure .autoteam directory exists
	if err := g.fileOps.EnsureDirectory(config.AutoTeamDir, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create .autoteam directory: %w", err)
	}

	// Ensure agents directories exist
	if err := g.createAgentDirectories(cfg); err != nil {
		return fmt.Errorf("failed to create agent directories: %w", err)
	}

	// Generate compose.yaml in .autoteam directory
	if err := g.generateFile("compose.yaml.tmpl", config.ComposeFilePath, cfg); err != nil {
		return fmt.Errorf("failed to generate compose.yaml: %w", err)
	}

	// Copy system entrypoints directory
	if err := g.copyEntrypointsDirectory(); err != nil {
		return fmt.Errorf("failed to copy entrypoints directory: %w", err)
	}

	return nil
}

func (g *Generator) copyEntrypointsDirectory() error {
	// Ensure agents directory exists
	if err := g.fileOps.EnsureDirectory(config.AgentsDir, config.DirPerm); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Remove existing directory if it exists
	if err := g.fileOps.RemoveIfExists(config.LocalEntrypointsPath); err != nil {
		return fmt.Errorf("failed to remove existing entrypoints directory: %w", err)
	}

	// Check if system entrypoints directory exists
	if !g.fileOps.DirectoryExists(config.SystemEntrypointsDir) {
		// Create a temporary directory with a helpful message
		if err := g.fileOps.EnsureDirectory(config.LocalEntrypointsPath, config.DirPerm); err != nil {
			return fmt.Errorf("failed to create temporary entrypoints directory: %w", err)
		}

		readmePath := filepath.Join(config.LocalEntrypointsPath, config.ReadmeFile)
		readmeContent := `# AutoTeam Entrypoint Binaries

This directory should contain entrypoint binaries for different platforms.

To install the entrypoint binaries system-wide, run:
` + "```bash" + `
autoteam --install-entrypoints
` + "```" + `

This will:
1. Install entrypoint binaries for all supported platforms to ` + config.SystemEntrypointsDir + `
2. Copy the binaries to this local directory during generation

Supported platforms:
- linux-amd64
- linux-arm64  
- darwin-amd64
- darwin-arm64
`

		if err := g.fileOps.WriteFileIfNotExists(readmePath, []byte(readmeContent), config.ReadmePerm); err != nil {
			return fmt.Errorf("failed to create README file: %w", err)
		}

		return nil
	}

	// Copy system entrypoints directory to local directory
	return g.fileOps.CopyDirectory(config.SystemEntrypointsDir, config.LocalEntrypointsPath)
}

func (g *Generator) createAgentDirectories(cfg *config.Config) error {
	for _, agent := range cfg.Agents {
		normalizedName := agent.GetNormalizedName()
		if err := g.fileOps.CreateAgentDirectoryStructure(normalizedName); err != nil {
			return fmt.Errorf("failed to create directory structure for agent %s (normalized: %s): %w", agent.Name, normalizedName, err)
		}
	}

	return nil
}

func (g *Generator) generateFile(templateFile, outputFile string, cfg *config.Config) error {
	// Create template data with agents that have effective settings
	templateData := struct {
		*config.Config
		AgentsWithSettings []config.AgentWithSettings
	}{
		Config:             cfg,
		AgentsWithSettings: cfg.GetAllAgentsWithEffectiveSettings(),
	}

	// Get template functions
	funcMap := GetTemplateFunctions()

	templatePath := filepath.Join("templates", templateFile)

	// Try embedded template first
	templateContent, err := templateFS.ReadFile(templatePath)
	if err != nil {
		// Fall back to external file for testing
		externalPath := filepath.Join("templates", templateFile)
		externalTmpl, parseErr := template.New(templateFile).Funcs(funcMap).ParseFiles(externalPath)
		if parseErr != nil {
			return fmt.Errorf("failed to read embedded template %s and external template %s: %w", templatePath, externalPath, parseErr)
		}

		output, createErr := os.Create(outputFile)
		if createErr != nil {
			return fmt.Errorf("failed to create output file %s: %w", outputFile, createErr)
		}
		defer output.Close()

		if execErr := externalTmpl.Execute(output, templateData); execErr != nil {
			return fmt.Errorf("failed to execute template %s: %w", templateFile, execErr)
		}
		return nil
	}

	tmpl, err := template.New(templateFile).Funcs(funcMap).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateFile, err)
	}

	output, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFile, err)
	}
	defer output.Close()

	if err := tmpl.Execute(output, templateData); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateFile, err)
	}

	return nil
}
