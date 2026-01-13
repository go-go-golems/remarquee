package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"dagger.io/dagger"
)

func main() {
	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		log.Fatalf("connect dagger: %v", err)
	}
	defer func() { _ = client.Close() }()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("getwd: %v", err)
	}

	repoRoot, err := findRepoRoot(wd)
	if err != nil {
		log.Fatalf("find repo root: %v", err)
	}

	frontendDir := filepath.Join(repoRoot, "cmd", "remarquee-ui", "frontend")
	if env := os.Getenv("RMQ_WEB_DIR"); env != "" {
		frontendDir = env
	}
	distOut := filepath.Join(frontendDir, "dist")
	if env := os.Getenv("RMQ_WEB_DIST"); env != "" {
		distOut = env
	}

	if err := os.RemoveAll(distOut); err != nil {
		log.Fatalf("remove dist: %v", err)
	}

	image := os.Getenv("WEB_BUILDER_IMAGE")
	if image == "" {
		image = "node:22"
	}

	webDir := client.Host().Directory(frontendDir)
	ctr := client.Container().
		From(image).
		WithWorkdir("/src").
		WithMountedDirectory("/src", webDir)

	installCmd := "npm ci"
	if _, err := os.Stat(filepath.Join(frontendDir, "package-lock.json")); err != nil {
		installCmd = "npm install"
	}

	ctr = ctr.
		WithExec([]string{"sh", "-lc", "node --version"}).
		WithExec([]string{"sh", "-lc", "npm --version"}).
		WithExec([]string{"sh", "-lc", installCmd}).
		WithExec([]string{"sh", "-lc", "npm run build"})

	dist := ctr.Directory("/src/dist")
	if _, err := dist.Export(ctx, distOut); err != nil {
		log.Fatalf("export dist: %v", err)
	}
	log.Printf("exported web dist to %s", distOut)
}

func findRepoRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", start)
		}
		dir = parent
	}
}
