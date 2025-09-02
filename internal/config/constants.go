package config

// System paths and directories
const (
	// SystemBinDir is the system-wide installation directory for all binaries (entrypoints, MCP servers, etc.)
	SystemBinDir = "/opt/autoteam/bin"

	// AutoTeamDir is the main .autoteam directory for all generated files
	AutoTeamDir = ".autoteam"

	// LocalBinPath is the local path where binaries are copied during generation
	LocalBinPath = AutoTeamDir + "/bin"

	// WorkersBaseDir is the base directory pattern for worker-specific directories
	// Actual path will be: .autoteam/{team_name}/workers
	WorkersBaseDir = AutoTeamDir + "/%s/workers"

	// ControlPlaneBaseDir is the base directory pattern for control-plane config
	// Actual path will be: .autoteam/{team_name}/control-plane
	ControlPlaneBaseDir = AutoTeamDir + "/%s/control-plane"

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
