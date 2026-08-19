package bootstrap

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveIaCCommandRejectsUnsupportedValue(t *testing.T) {
	_, err := resolveIaCCommand("pulumi")

	require.EqualError(t, err, `unsupported --iac-command "pulumi"; expected auto, terraform, or tofu`)
}

func TestCappedBufferLimitsCapturedOutput(t *testing.T) {
	buffer := cappedBuffer{limit: 4}

	written, err := buffer.Write([]byte("secret"))

	require.NoError(t, err)
	require.Equal(t, len("secret"), written)
	require.Equal(t, "secr", buffer.String())
	require.True(t, buffer.exceeded)
	require.False(t, strings.Contains(buffer.String(), "et"))
}
