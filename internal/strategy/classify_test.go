package strategy

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/require"
)

func TestClassify_prebuiltPublic(t *testing.T) {
	s := types.ServiceConfig{Image: "nginx:alpine"}
	r := Classify(File{}, "web", s)
	require.Equal(t, PrebuiltPublic, r.Kind)
}

func TestClassify_ecrPrivate(t *testing.T) {
	s := types.ServiceConfig{Image: "123456789012.dkr.ecr.us-east-1.amazonaws.com/app:v1"}
	r := Classify(File{}, "app", s)
	require.Equal(t, PrivateImage, r.Kind)
}

func TestClassify_buildLocal(t *testing.T) {
	s := types.ServiceConfig{Build: &types.BuildConfig{Context: "."}}
	r := Classify(File{}, "api", s)
	require.Equal(t, BuildLocal, r.Kind)
}

func TestClassify_strategyPrivate(t *testing.T) {
	sf := File{Services: map[string]ServiceRule{"api": {Sourcing: "private"}}}
	s := types.ServiceConfig{Image: "nginx:alpine"}
	r := Classify(sf, "api", s)
	require.Equal(t, PrivateImage, r.Kind)
}

func TestClassify_labelOverrides(t *testing.T) {
	s := types.ServiceConfig{
		Image:  "nginx:alpine",
		Labels: types.Labels{"zeabur.sourcing": "private"},
	}
	r := Classify(File{}, "web", s)
	require.Equal(t, PrivateImage, r.Kind)
}
