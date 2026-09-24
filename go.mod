module github.com/gmyilma/gardener-mcp-server

// Minimum language version. Set by github.com/gardener/gardener/pkg/apis,
// which declares go 1.26.0 (architecture-review.md §5, row 13).
go 1.26.0

// Exact toolchain used to build and test. With GOTOOLCHAIN=auto (the default,
// also set in .mise.toml) the Go command downloads this version automatically,
// so contributors get a reproducible build without installing anything by hand.
toolchain go1.26.8

// Dev tools are deliberately not listed as `tool` directives here. Keeping them
// in .mise.toml leaves this file describing only what ships in the binary,
// which is what makes the dependency graph auditable (architecture-review.md §7).

require (
	github.com/modelcontextprotocol/go-sdk v1.8.0
	golang.org/x/text v0.42.0
)

require (
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)
