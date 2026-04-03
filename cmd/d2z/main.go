// Command d2z loads Docker Compose projects, validates prerequisites, analyzes services, and renders Zeabur-oriented YAML.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "d2z: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf(`usage: d2z <check|analyze|render|clone> [flags]

commands:
  check    verify tools and compose file(s)
  analyze  print classification, depends_on, healthchecks
  render   write Zeabur-oriented YAML to stdout or -out
  clone    git clone or pull REPO_URL into WORK_DIR`)
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "check":
		return runCheck(ctx, rest)
	case "analyze":
		return runAnalyze(ctx, rest)
	case "render":
		return runRender(ctx, rest)
	case "clone":
		return runClone(ctx, rest)
	case "help", "-h", "--help":
		return run(ctx, nil)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func defaultComposeFiles(workDir string) []string {
	candidates := []string{
		filepath.Join(workDir, "compose.yaml"),
		filepath.Join(workDir, "compose.yml"),
		filepath.Join(workDir, "docker-compose.yaml"),
		filepath.Join(workDir, "docker-compose.yml"),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return []string{c}
		}
	}
	return []string{filepath.Join(workDir, "docker-compose.yml")}
}
