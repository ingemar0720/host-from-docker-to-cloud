package composeproj

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/graph"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"

	"github.com/ingemar0720/host-from-docker-to-cloud/internal/envmap"
)

// Load reads Compose files relative to workDir and returns a validated project.
func Load(ctx context.Context, workDir string, composeFiles []string) (*types.Project, error) {
	if len(composeFiles) == 0 {
		return nil, fmt.Errorf("no compose files specified")
	}
	absDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(composeFiles))
	for _, f := range composeFiles {
		var p string
		switch {
		case filepath.IsAbs(f):
			p = f
		default:
			if _, err := os.Stat(f); err == nil {
				ap, err := filepath.Abs(f)
				if err != nil {
					return nil, err
				}
				paths = append(paths, ap)
				continue
			}
			p = filepath.Join(absDir, f)
		}
		ap, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		paths = append(paths, ap)
	}
	details := types.ConfigDetails{
		WorkingDir:  absDir,
		ConfigFiles: types.ToConfigFiles(paths),
		Environment: envmap.FromOS(),
	}
	p, err := loader.LoadWithContext(ctx, details)
	if err != nil {
		return nil, err
	}
	if err := graph.CheckCycle(p); err != nil {
		return nil, err
	}
	return p, nil
}
