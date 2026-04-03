package envmap

import (
	"os"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

// FromOS returns the current process environment as compose types.Mapping.
func FromOS() types.Mapping {
	m := types.Mapping{}
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if ok {
			m[k] = v
		}
	}
	return m
}
