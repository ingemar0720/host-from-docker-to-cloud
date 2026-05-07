package sourcemode

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ingemar0720/host-from-docker-to-cloud/internal/composeproj"
)

func writeCompose(t *testing.T, dir, body string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(body), 0o644))
}

func TestValidate_acceptsBuildOrImage(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, `
name: valid
services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
  db:
    image: postgres:16-alpine
`)
	p, err := composeproj.Load(context.Background(), dir, []string{"docker-compose.yml"})
	require.NoError(t, err)
	require.Empty(t, Validate(p))
}

func TestValidate_rejectsBothModes(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, `
name: both-mode
services:
  api:
    image: myorg/api:latest
    build:
      context: .
      dockerfile: Dockerfile
`)
	p, err := composeproj.Load(context.Background(), dir, []string{"docker-compose.yml"})
	require.NoError(t, err)
	errs := Validate(p)
	require.Len(t, errs, 1)
	require.Contains(t, errs[0].Error(), "both build and image are set")
}
