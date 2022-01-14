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

	os.Setenv("DRYCC_REDIS_PASSWORD", "password")
	os.Setenv("DRYCC_REDIS_PIPELINE_LENGTH", "1")
	os.Setenv("DRYCC_REDIS_PIPELINE_TIMEOUT_SECONDS", "2")

	c, err := parseConfig("foo")
	assert.NoError(t, err, "error parsing config")
	assert.Equal(t, c.Addrs, original.Addrs)
	assert.Equal(t, c.Password, "password")
	assert.Equal(t, c.PipelineLength, 1)
	assert.Equal(t, c.PipelineTimeoutSeconds, 2)

	os.Setenv("DRYCC_REDIS_PASSWORD", original.Password)
	os.Setenv("DRYCC_REDIS_PIPELINE_LENGTH", fmt.Sprint(original.PipelineLength))
	os.Setenv("DRYCC_REDIS_PIPELINE_TIMEOUT_SECONDS", fmt.Sprint(original.PipelineTimeoutSeconds))
}
