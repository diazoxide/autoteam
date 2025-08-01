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

	// CodebaseSubdir is the subdirectory name for agent codebase
	CodebaseSubdir = "codebase"
)

// File names and extensions
const (
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

	// ExecutablePerm is the permission for executable files
	ExecutablePerm = 0755

	// ReadmePerm is the permission for README files
	ReadmePerm = 0644
)
