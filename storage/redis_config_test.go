package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	addrs := os.Getenv("DRYCC_LOGGER_REDIS_ADDRS")
	password := os.Getenv("DRYCC_LOGGER_REDIS_PASSWORD")
	pipelineLength := os.Getenv("DRYCC_LOGGER_REDIS_PIPELINE_LENGTH")
	pipelineTimeoutSeconds := os.Getenv("DRYCC_LOGGER_REDIS_PIPELINE_TIMEOUT_SECONDS")

	os.Setenv("DRYCC_LOGGER_REDIS_PASSWORD", "password")
	os.Setenv("DRYCC_LOGGER_REDIS_PIPELINE_LENGTH", "1")
	os.Setenv("DRYCC_LOGGER_REDIS_PIPELINE_TIMEOUT_SECONDS", "2")

	c, err := parseConfig("foo")
	assert.NoError(t, err, "error parsing config")
	assert.Equal(t, c.Addrs, addrs)
	assert.Equal(t, c.Password, "password")
	assert.Equal(t, c.PipelineLength, 1)
	assert.Equal(t, c.PipelineTimeoutSeconds, 2)

	os.Setenv("DRYCC_LOGGER_REDIS_PASSWORD", password)
	os.Setenv("DRYCC_LOGGER_REDIS_PIPELINE_LENGTH", pipelineLength)
	os.Setenv("DRYCC_LOGGER_REDIS_PIPELINE_TIMEOUT_SECONDS", pipelineTimeoutSeconds)
}
