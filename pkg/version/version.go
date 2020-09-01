package version

// Dynamically inserted at build time. See `ldflags` in .goreleaser.yml.
var (
	Version  = "unset"
	Revision = "unset"
)
