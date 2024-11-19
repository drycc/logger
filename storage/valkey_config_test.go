package storage

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	original, err := parseConfig("foo")
	assert.NoError(t, err, "error parsing config")
	valkeyURL := "redis://test.redis.com:16379/1"
	os.Setenv("DRYCC_VALKEY_URL", valkeyURL)
	os.Setenv("DRYCC_VALKEY_PIPELINE_LENGTH", "1")
	os.Setenv("DRYCC_VALKEY_PIPELINE_TIMEOUT_SECONDS", "2")

	c, err := parseConfig("foo")
	assert.NoError(t, err, "error parsing config")
	assert.Equal(t, c.URL, valkeyURL)
	assert.Equal(t, c.PipelineLength, 1)
	assert.Equal(t, c.PipelineTimeoutSeconds, 2)

	os.Setenv("DRYCC_VALKEY_URL", original.URL)
	os.Setenv("DRYCC_VALKEY_PIPELINE_LENGTH", fmt.Sprint(original.PipelineLength))
	os.Setenv("DRYCC_VALKEY_PIPELINE_TIMEOUT_SECONDS", fmt.Sprint(original.PipelineTimeoutSeconds))
}
