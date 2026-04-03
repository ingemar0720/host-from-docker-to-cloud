package composeproj

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeCompose(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestLoad_validProject(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "docker-compose.yml", `
name: test-proj
services:
  db:
    image: postgres:16-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
  api:
    image: nginx:alpine
    depends_on:
      db:
        condition: service_healthy
`)

	p, err := Load(context.Background(), dir, []string{"docker-compose.yml"})
	require.NoError(t, err)
	require.Equal(t, "test-proj", p.Name)
	require.ElementsMatch(t, []string{"api", "db"}, p.ServiceNames())
	require.Contains(t, p.Services["api"].DependsOn, "db")
	require.Equal(t, "service_healthy", p.Services["api"].DependsOn["db"].Condition)
}

func TestLoad_detectsDependencyCycle(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "docker-compose.yml", `
name: cyclic
services:
  a:
    image: alpine:3.20
    depends_on:
      - b
  b:
    image: alpine:3.20
    depends_on:
      - a
`)

	_, err := Load(context.Background(), dir, []string{"docker-compose.yml"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cycle")
}

func TestLoad_noComposeFiles(t *testing.T) {
	_, err := Load(context.Background(), t.TempDir(), nil)
	require.Error(t, err)
}

func TestLoad_resolvePathRelativeToWorkdir(t *testing.T) {
	dir := t.TempDir()
	writeCompose(t, dir, "compose.yaml", `
name: wd
services:
  web:
    image: nginx:alpine
`)
	p, err := Load(context.Background(), dir, []string{"compose.yaml"})
	require.NoError(t, err)
	require.Equal(t, "wd", p.Name)
}
