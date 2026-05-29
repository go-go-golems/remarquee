// Package remarquee hosts module-root generation directives.
//
// The repository's reusable library code lives under ./pkg; this root package is
// intentionally separate so go generate ./... can run the logcopter generator
// from the module root while scanning ./cmd/... and ./pkg/....
package remarquee

//go:generate go tool logcopter-gen -include-main -var zlog -area-prefix go-go-golems.remarquee -strip-prefix github.com/go-go-golems/remarquee ./cmd/... ./pkg/...
