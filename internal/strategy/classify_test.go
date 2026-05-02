package strategy

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/require"
)

func TestClassify_imagePublic(t *testing.T) {
	s := types.ServiceConfig{Image: "nginx:alpine"}
	r := Classify(File{}, "web", s)
	require.Equal(t, ImageDockerHubPublic, r.Kind)
}

func TestClassify_buildFromSource(t *testing.T) {
	s := types.ServiceConfig{Build: &types.BuildConfig{Context: "."}}
	r := Classify(File{}, "app", s)
	require.Equal(t, BuildFromSource, r.Kind)
}

func TestClassify_strategyImagePrivate(t *testing.T) {
	sf := File{Services: map[string]ServiceRule{"api": {Sourcing: "image-private"}}}
	s := types.ServiceConfig{Image: "myorg/private-app:latest"}
	r := Classify(sf, "api", s)
	require.Equal(t, ImageDockerHubPrivate, r.Kind)
}

func TestClassify_strategyBuild(t *testing.T) {
	sf := File{Services: map[string]ServiceRule{"api": {Sourcing: "build"}}}
	s := types.ServiceConfig{Image: "nginx:alpine", Build: &types.BuildConfig{Context: "."}}
	r := Classify(sf, "api", s)
	require.Equal(t, BuildFromSource, r.Kind)
}

func TestClassify_labelImagePrivate(t *testing.T) {
	s := types.ServiceConfig{
		Image:  "nginx:alpine",
		Labels: types.Labels{"zeabur.sourcing": "image-private"},
	}
	r := Classify(File{}, "web", s)
	require.Equal(t, ImageDockerHubPrivate, r.Kind)
}
