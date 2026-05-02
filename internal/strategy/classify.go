package strategy

import (
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

// Kind describes deployment sourcing for a service.
type Kind string

const (
	BuildFromSource       Kind = "build-from-source"
	ImageDockerHubPublic  Kind = "image-dockerhub-public"
	ImageDockerHubPrivate Kind = "image-dockerhub-private"
)

// Result is per-service classification with rationale.
type Result struct {
	Service string
	Kind    Kind
	Reason  string
}

const (
	labelSourcing = "zeabur.sourcing"
)

// Classify applies strategy file, compose labels, and heuristics.
func Classify(sf File, name string, svc types.ServiceConfig) Result {
	rule := ""
	if sf.Services != nil {
		if sr, ok := sf.Services[name]; ok {
			rule = strings.TrimSpace(strings.ToLower(sr.Sourcing))
		}
	}
	if rule == "" && svc.Labels != nil {
		if v, ok := svc.Labels[labelSourcing]; ok {
			rule = strings.TrimSpace(strings.ToLower(v))
		}
	}

	img := strings.TrimSpace(svc.Image)
	hasBuild := svc.Build != nil

	switch rule {
	case "build", "source":
		return Result{name, BuildFromSource, "strategy/label: sourcing=build → build from Dockerfile"}
	case "image-public", "public":
		return Result{name, ImageDockerHubPublic, "strategy/label: sourcing=image-public → pull public image"}
	case "image-private", "private":
		return Result{name, ImageDockerHubPrivate, "strategy/label: sourcing=image-private → pull private Docker Hub image"}
	case "auto", "":
		return classifyAuto(name, img, hasBuild)
	default:
		r := classifyAuto(name, img, hasBuild)
		r.Reason = "unknown strategy " + rule + "; auto: " + r.Reason
		return r
	}
}

func classifyAuto(name, image string, hasBuild bool) Result {
	if hasBuild {
		return Result{name, BuildFromSource, "build: set → build from source"}
	}
	if image == "" {
		return Result{name, BuildFromSource, "no image; build assumed"}
	}
	return Result{name, ImageDockerHubPublic, "image set; assumed public unless overridden to image-private"}
}
