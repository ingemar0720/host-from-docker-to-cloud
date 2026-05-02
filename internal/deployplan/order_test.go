package deployplan

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ingemar0720/host-from-docker-to-cloud/internal/composeproj"
)

func TestOrder_dependencyBeforeDependent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(`
name: ord
services:
  api:
    image: nginx:alpine
    depends_on:
      - db
  db:
    image: postgres:16-alpine
`), 0o644))

	p, err := composeproj.Load(context.Background(), dir, []string{"docker-compose.yml"})
	require.NoError(t, err)
	order, err := Order(p)
	require.NoError(t, err)
	require.Equal(t, []string{"db", "api"}, order)
}

func TestOrder_lexicalTieBreak(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(`
name: lexical
services:
  z:
    image: nginx:alpine
  a:
    image: nginx:alpine
  m:
    image: nginx:alpine
`), 0o644))

	p, err := composeproj.Load(context.Background(), dir, []string{"docker-compose.yml"})
	require.NoError(t, err)
	order, err := Order(p)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "m", "z"}, order)
}
