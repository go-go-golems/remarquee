package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
)

const (
	defaultBuilderImage = "node:22"
	defaultPNPMVersion  = "10.15.1"
	pnpmStorePath       = "/pnpm/store"
)

func main() {
	ctx := context.Background()

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

	pnpmVersion := packageManagerPNPMVersion(frontendDir)
	if pnpmVersion == "" {
		pnpmVersion = defaultPNPMVersion
	}

	if forceLocal := os.Getenv("BUILD_WEB_LOCAL"); forceLocal != "" {
		log.Printf("BUILD_WEB_LOCAL=%s: building web UI with local pnpm", forceLocal)
		if err := buildLocal(ctx, frontendDir, distOut, pnpmVersion); err != nil {
			log.Fatalf("local web build failed: %v", err)
		}
		return
	}

	if err := buildWithDagger(ctx, frontendDir, distOut, pnpmVersion); err != nil {
		log.Printf("dagger web build failed: %v", err)
		log.Printf("falling back to local pnpm build; set BUILD_WEB_LOCAL=1 to force this path")
		if err := buildLocal(ctx, frontendDir, distOut, pnpmVersion); err != nil {
			log.Fatalf("local web build failed: %v", err)
		}
	}
}

func buildWithDagger(ctx context.Context, frontendDir string, distOut string, pnpmVersion string) error {
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return fmt.Errorf("connect dagger: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := os.RemoveAll(distOut); err != nil {
		return fmt.Errorf("remove dist: %w", err)
	}

	image := os.Getenv("WEB_BUILDER_IMAGE")
	if image == "" {
		image = defaultBuilderImage
	}

	webDir := client.Host().Directory(frontendDir, dagger.HostDirectoryOpts{
		Exclude: []string{"node_modules", ".pnpm-store", "dist"},
	})
	pnpmStore := client.CacheVolume("remarquee-ui-pnpm-store")

	ctr := client.Container().
		From(image).
		WithWorkdir("/src").
		WithMountedDirectory("/src", webDir).
		WithMountedCache(pnpmStorePath, pnpmStore).
		WithEnvVariable("COREPACK_ENABLE_DOWNLOAD_PROMPT", "0").
		WithEnvVariable("CI", "1").
		WithExec([]string{"sh", "-lc", "node --version"}).
		WithExec([]string{"sh", "-lc", "corepack enable"}).
		WithExec([]string{"sh", "-lc", fmt.Sprintf("corepack prepare pnpm@%s --activate", shellQuote(pnpmVersion))}).
		WithExec([]string{"sh", "-lc", "pnpm --version"}).
		WithExec([]string{"sh", "-lc", fmt.Sprintf("pnpm config set store-dir %s", shellQuote(pnpmStorePath))}).
		WithExec([]string{"sh", "-lc", "pnpm install --frozen-lockfile"}).
		WithExec([]string{"sh", "-lc", "pnpm run build"})

	if _, err := ctr.Directory("/src/dist").Export(ctx, distOut); err != nil {
		return fmt.Errorf("export dist: %w", err)
	}
	log.Printf("exported web dist to %s", distOut)
	return nil
}

func buildLocal(ctx context.Context, frontendDir string, distOut string, pnpmVersion string) error {
	defaultDist := filepath.Join(frontendDir, "dist")
	if err := os.RemoveAll(defaultDist); err != nil {
		return fmt.Errorf("remove default dist: %w", err)
	}
	if filepath.Clean(distOut) != filepath.Clean(defaultDist) {
		if err := os.RemoveAll(distOut); err != nil {
			return fmt.Errorf("remove custom dist: %w", err)
		}
	}

	commands := [][]string{
		{"corepack", "enable"},
		{"corepack", "prepare", "pnpm@" + pnpmVersion, "--activate"},
		{"pnpm", "config", "set", "store-dir", filepath.Join(frontendDir, ".pnpm-store")},
		{"pnpm", "install", "--frozen-lockfile"},
		{"pnpm", "run", "build"},
	}
	for _, argv := range commands {
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Dir = frontendDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", strings.Join(argv, " "), err)
		}
	}
	if filepath.Clean(distOut) != filepath.Clean(defaultDist) {
		if err := copyDir(defaultDist, distOut); err != nil {
			return fmt.Errorf("copy dist to requested output: %w", err)
		}
	}
	log.Printf("exported web dist to %s", distOut)
	return nil
}

func copyDir(src string, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source is not a directory: %s", src)
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target)
	})
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func packageManagerPNPMVersion(frontendDir string) string {
	b, err := os.ReadFile(filepath.Join(frontendDir, "package.json"))
	if err != nil {
		return ""
	}
	needle := `"packageManager"`
	idx := strings.Index(string(b), needle)
	if idx < 0 {
		return ""
	}
	rest := string(b)[idx+len(needle):]
	pnpmIdx := strings.Index(rest, "pnpm@")
	if pnpmIdx < 0 {
		return ""
	}
	version := rest[pnpmIdx+len("pnpm@"):]
	end := strings.IndexAny(version, `"' \n\r\t`)
	if end >= 0 {
		version = version[:end]
	}
	return strings.TrimSpace(version)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
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
