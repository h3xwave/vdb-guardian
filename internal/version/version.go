package version

// InfoValue describes build metadata that can be shared by the CLI, server, and
// diagnostics endpoints. Keeping this structure in a dedicated package avoids
// hardcoding project identity in multiple command entrypoints.
type InfoValue struct {
	// Name is the stable project identifier used in logs, CLI output, and reports.
	Name string
	// Version is the semantic or development version displayed to operators.
	Version string
}

const developmentVersion = "dev"

// Info returns the current project metadata for user-facing commands and service
// diagnostics. The function currently reports a development version because the
// repository scaffold does not yet have release-time linker injection.
func Info() InfoValue {
	return InfoValue{Name: "vdb-guardian", Version: developmentVersion}
}
