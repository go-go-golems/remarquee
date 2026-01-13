package main

import (
	"embed"
	"io/fs"
)

//go:embed frontend/dist
var frontendDist embed.FS

// GetFrontendFS returns the embedded frontend filesystem
// In production mode, this serves the static assets from frontend/dist/
func GetFrontendFS() (fs.FS, error) {
	return fs.Sub(frontendDist, "frontend/dist")
}
