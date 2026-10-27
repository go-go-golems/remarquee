package serve

import (
	"embed"
	"io/fs"
)

//go:embed frontend
var frontendFS embed.FS

// frontendRoot returns the embedded frontend directory as an fs.FS rooted at it.
func frontendRoot() (fs.FS, error) {
	return fs.Sub(frontendFS, "frontend")
}
