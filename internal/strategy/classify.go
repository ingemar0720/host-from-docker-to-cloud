package strategy

import (
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

// Kind describes deployment sourcing for a service.
type Kind string

const (
	PrebuiltPublic  Kind = "prebuilt-public"
	BuildLocal      Kind = "build-local"
	PrivateImage    Kind = "private-image"
	PrivateBuildECR Kind = "private-build-ecr"
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
	case "private":
		if hasBuild {
			return Result{name, PrivateBuildECR, "strategy/label: sourcing=private with build → push to ECR"}
		}
		return Result{name, PrivateImage, "strategy/label: sourcing=private with image → use private registry / ECR pull"}
	case "public":
		if hasBuild {
			return Result{name, BuildLocal, "strategy/label: sourcing=public with build → build from Dockerfile"}
		}
		return Result{name, PrebuiltPublic, "strategy/label: sourcing=public → prebuilt image"}
	case "auto", "":
		return classifyAuto(name, img, hasBuild)
	default:
		r := classifyAuto(name, img, hasBuild)
		r.Reason = "unknown strategy " + rule + "; auto: " + r.Reason
		return r
	}
}

func classifyAuto(name, image string, hasBuild bool) Result {
	if hasBuild && isLikelyPrivateBuild(image) {
		return Result{name, PrivateBuildECR, "build context + private-style image reference"}
	}
	if hasBuild {
		return Result{name, BuildLocal, "build: set, no public-only image"}
	}
	if image == "" {
		return Result{name, BuildLocal, "no image; build assumed"}
	}
	if isPrivateRegistryImage(image) {
		return Result{name, PrivateImage, "image host looks private (ECR/custom registry)"}
	}
	return Result{name, PrebuiltPublic, "image from public-style reference"}
}

func isPrivateRegistryImage(image string) bool {
	l := strings.ToLower(image)
	switch {
	case strings.Contains(l, ".dkr.ecr.") && strings.Contains(l, ".amazonaws.com"):
		return true
	case strings.HasPrefix(l, "localhost"), strings.HasPrefix(l, "127.0.0.1"):
		return true
	case strings.Contains(l, ":5000/"):
		return true
	case strings.HasPrefix(l, "gcr.io/"), strings.HasPrefix(l, "asia.gcr.io/"), strings.HasPrefix(l, "eu.gcr.io/"), strings.HasPrefix(l, "us.gcr.io/"):
		// Could be public; treat as org registry — user overrides with label.
		return false
	case strings.HasPrefix(l, "ghcr.io/"):
		return false
	default:
		// Host:port/repo without docker.io
		if idx := strings.Index(l, "/"); idx > 0 {
			host := l[:idx]
			if strings.Contains(host, ".") && !strings.Contains(host, "docker.io") {
				// e.g. registry.example.com/myimg
				if !strings.HasSuffix(host, "docker.io") && host != "docker.io" {
					return true
				}
			}
		}
		return false
	}
}

func isLikelyPrivateBuild(image string) bool {
	return image != "" && isPrivateRegistryImage(image)
}
