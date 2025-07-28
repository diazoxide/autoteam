package generator

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"auto-team/internal/config"
)

//go:embed templates/*
var templateFS embed.FS

type Generator struct {}

func New() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateCompose(cfg *config.Config) error {
	// Ensure agents directories exist
	if err := g.createAgentDirectories(cfg); err != nil {
		return fmt.Errorf("failed to create agent directories: %w", err)
	}

	// Generate compose.yaml
	if err := g.generateFile("compose.yaml.tmpl", "compose.yaml", cfg); err != nil {
		return fmt.Errorf("failed to generate compose.yaml: %w", err)
	}

	// Generate entrypoint.sh
	if err := g.generateFile("entrypoint.sh.tmpl", "entrypoint.sh", cfg); err != nil {
		return fmt.Errorf("failed to generate entrypoint.sh: %w", err)
	}

	// Make entrypoint.sh executable
	if err := os.Chmod("entrypoint.sh", 0755); err != nil {
		return fmt.Errorf("failed to make entrypoint.sh executable: %w", err)
	}

	// Ensure shared directory exists
	if err := os.MkdirAll("shared", 0755); err != nil {
		return fmt.Errorf("failed to create shared directory: %w", err)
	}

	return nil
}

func (g *Generator) createAgentDirectories(cfg *config.Config) error {
	for _, agent := range cfg.Agents {
		agentDir := filepath.Join("agents", agent.Name)
		codebaseDir := filepath.Join(agentDir, "codebase")
		claudeDir := filepath.Join(agentDir, "claude")

		// Create agent directories
		if err := os.MkdirAll(codebaseDir, 0755); err != nil {
			return fmt.Errorf("failed to create codebase directory for agent %s: %w", agent.Name, err)
		}

		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			return fmt.Errorf("failed to create claude directory for agent %s: %w", agent.Name, err)
		}

		// Create empty .claude and .claude.json files if they don't exist
		claudeConfigPath := filepath.Join(claudeDir, ".claude")
		if _, err := os.Stat(claudeConfigPath); os.IsNotExist(err) {
			if err := os.WriteFile(claudeConfigPath, []byte(""), 0600); err != nil {
				return fmt.Errorf("failed to create .claude file for agent %s: %w", agent.Name, err)
			}
		}

		claudeJSONPath := filepath.Join(claudeDir, ".claude.json")
		if _, err := os.Stat(claudeJSONPath); os.IsNotExist(err) {
			if err := os.WriteFile(claudeJSONPath, []byte("{}"), 0600); err != nil {
				return fmt.Errorf("failed to create .claude.json file for agent %s: %w", agent.Name, err)
			}
		}
	}

	return nil
}

func (g *Generator) generateFile(templateFile, outputFile string, cfg *config.Config) error {
	templatePath := filepath.Join("templates", templateFile)

	// Try embedded template first
	templateContent, err := templateFS.ReadFile(templatePath)
	if err != nil {
		// Fall back to external file for testing
		externalPath := filepath.Join("templates", templateFile)
		tmpl, err := template.ParseFiles(externalPath)
		if err != nil {
			return fmt.Errorf("failed to read embedded template %s and external template %s: %w", templatePath, externalPath, err)
		}
		
		output, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file %s: %w", outputFile, err)
		}
		defer output.Close()

		if err := tmpl.Execute(output, cfg); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", templateFile, err)
		}
		return nil
	}

	tmpl, err := template.New(templateFile).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateFile, err)
	}

	output, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputFile, err)
	}
	defer output.Close()

	if err := tmpl.Execute(output, cfg); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateFile, err)
	}

	return nil
}
