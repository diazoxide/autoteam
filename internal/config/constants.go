package config

// System paths and directories
const (
	// SystemEntrypointsDir is the system-wide installation directory for entrypoint binaries
	SystemEntrypointsDir = "/opt/autoteam/entrypoints"

	// AutoTeamDir is the main .autoteam directory for all generated files
	AutoTeamDir = ".autoteam"

	// LocalEntrypointsPath is the local path where entrypoints are copied during generation
	LocalEntrypointsPath = AutoTeamDir + "/entrypoints"

	// AgentsDir is the base directory for all agent-specific directories
	AgentsDir = AutoTeamDir + "/agents"

	// SharedDir is the directory for shared configuration files
	SharedDir = AutoTeamDir + "/shared"

	// CodebaseSubdir is the subdirectory name for agent codebase
	CodebaseSubdir = "codebase"

	// ClaudeSubdir is the subdirectory name for Claude configuration
	ClaudeSubdir = "claude"
)

// File names and extensions
const (
	// ClaudeConfigFile is the name of the Claude configuration file
	ClaudeConfigFile = ".claude"

	// ClaudeJSONFile is the name of the Claude JSON configuration file
	ClaudeJSONFile = ".claude.json"

	// ComposeFile is the name of the Docker Compose file
	ComposeFile = "compose.yaml"

	// ComposeFilePath is the full path to the Docker Compose file in .autoteam directory
	ComposeFilePath = AutoTeamDir + "/compose.yaml"

	// EntrypointScript is the name of the entrypoint shell script
	EntrypointScript = "entrypoint.sh"

	// ReadmeFile is the name of README files
	ReadmeFile = "README.md"
)

// File permissions
const (
	// DirPerm is the default permission for directories
	DirPerm = 0755

	// ConfigFilePerm is the permission for configuration files
	ConfigFilePerm = 0600

	// ExecutablePerm is the permission for executable files
	ExecutablePerm = 0755

	// ReadmePerm is the permission for README files
	ReadmePerm = 0644
)
