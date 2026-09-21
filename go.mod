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

require golang.org/x/text v0.42.0
