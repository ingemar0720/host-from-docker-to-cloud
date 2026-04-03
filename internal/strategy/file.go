package strategy

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// File is optional zeabur.strategy.yaml (or path passed with -strategy).
type File struct {
	Services map[string]ServiceRule `yaml:"services"`
	ECR      ECRDefaults            `yaml:"ecr"`
}

type ServiceRule struct {
	Sourcing string `yaml:"sourcing"` // auto | public | private
}

type ECRDefaults struct {
	Registry        string `yaml:"registry"`
	RepositoryPrefix string `yaml:"repository_prefix"`
}

// LoadStrategyFile reads YAML from path. A non-empty path must refer to an existing file.
func LoadStrategyFile(path string) (File, error) {
	var f File
	if path == "" {
		return f, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return f, fmt.Errorf("read strategy file %s: %w", path, err)
	}
	if err := yaml.Unmarshal(b, &f); err != nil {
		return f, fmt.Errorf("strategy file %s: %w", path, err)
	}
	return f, nil
}

// LoadDefaultStrategyFile reads workdir/zeabur.strategy.yaml if it exists; otherwise returns an empty File.
func LoadDefaultStrategyFile(workdir string) (File, error) {
	path := filepath.Join(workdir, "zeabur.strategy.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, fmt.Errorf("read strategy file %s: %w", path, err)
	}
	var f File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return f, fmt.Errorf("strategy file %s: %w", path, err)
	}
	return f, nil
}
