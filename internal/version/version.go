package version

// Build-time variables (set by ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// GetVersion returns the formatted version string
func GetVersion() string {
	return Version
}

// GetBuildInfo returns build information
func GetBuildInfo() (version, buildTime, gitCommit string) {
	return Version, BuildTime, GitCommit
}
