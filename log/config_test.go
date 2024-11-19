package log

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStopTimeoutDuration(t *testing.T) {
	c := config{
		StopTimeoutSeconds: 60,
	}
	assert.Equal(t, c.stopTimeoutDuration(), time.Duration(c.StopTimeoutSeconds)*time.Second)
}

func TestParseConfig(t *testing.T) {
	original, err := parseConfig("foo")
	assert.NoError(t, err)

	os.Setenv("DRYCC_VALKEY_URL", "redis://127.0.0.1:6379")
	os.Setenv("DRYCC_VALKEY_STREAM", "log")
	os.Setenv("DRYCC_VALKEY_STREAM_GROUP", "logger")
	os.Setenv("DRYCC_VALKEY_STREAM_COUNT", "30")
	os.Setenv("DRYCC_VALKEY_STREAM_BLOCK", "30")
	os.Setenv("AGGREGATOR_STOP_TIMEOUT_SEC", "2")

	c, err := parseConfig("foo")
	assert.NoError(t, err)
	assert.Equal(t, c.ValkeyURL, "redis://127.0.0.1:6379")
	assert.Equal(t, c.ValkeyStream, "log")
	assert.Equal(t, c.ValkeyStreamGroup, "logger")
	assert.Equal(t, c.StopTimeoutSeconds, 2)

	os.Setenv("DRYCC_VALKEY_URL", original.ValkeyURL)
	os.Setenv("DRYCC_VALKEY_STREAM", original.ValkeyStream)
	os.Setenv("DRYCC_VALKEY_STREAM_GROUP", original.ValkeyStreamGroup)
	os.Setenv("AGGREGATOR_STOP_TIMEOUT_SEC", fmt.Sprint(original.StopTimeoutSeconds))
}
