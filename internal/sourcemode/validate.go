package sourcemode

import (
	"fmt"
	"sort"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

// Validate enforces explicit and predictable per-service source mode.
// Rules:
// - exactly one mode must be set: build or image
func Validate(project *types.Project) []error {
	names := project.ServiceNames()
	sort.Strings(names)

	var errs []error
	for _, name := range names {
		svc := project.Services[name]
		image := strings.TrimSpace(svc.Image)
		hasBuild := svc.Build != nil
		hasImage := image != ""

		switch {
		case hasBuild && hasImage:
			errs = append(errs, fmt.Errorf("service %q: both build and image are set; choose exactly one mode", name))
			continue
		case !hasBuild && !hasImage:
			errs = append(errs, fmt.Errorf("service %q: missing both build and image; choose one mode", name))
			continue
		}
	}
	return errs
}
