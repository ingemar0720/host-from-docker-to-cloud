package envmap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromOS_containsCurrentTestEnv(t *testing.T) {
	t.Setenv("D2Z_TEST_MARKER", "xyzzy")
	m := FromOS()
	require.Equal(t, "xyzzy", m["D2Z_TEST_MARKER"])
}
