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

	os.Setenv("DRYCC_REDIS_ADDRS", "127.0.0.1:6379")
	os.Setenv("DRYCC_REDIS_PASSWORD", "123456")
	os.Setenv("DRYCC_REDIS_STREAM", "log")
	os.Setenv("DRYCC_REDIS_STREAM_GROUP", "logger")
	os.Setenv("DRYCC_REDIS_STREAM_COUNT", "30")
	os.Setenv("DRYCC_REDIS_STREAM_BLOCK", "30")
	os.Setenv("AGGREGATOR_STOP_TIMEOUT_SEC", "2")

	c, err := parseConfig("foo")
	assert.NoError(t, err)
	assert.Equal(t, c.RedisAddrs, "127.0.0.1:6379")
	assert.Equal(t, c.RedisPassword, "123456")
	assert.Equal(t, c.RedisStream, "log")
	assert.Equal(t, c.RedisStreamGroup, "logger")
	assert.Equal(t, c.StopTimeoutSeconds, 2)

	os.Setenv("DRYCC_REDIS_ADDRS", original.RedisAddrs)
	os.Setenv("DRYCC_REDIS_PASSWORD", original.RedisPassword)
	os.Setenv("DRYCC_REDIS_STREAM", original.RedisStream)
	os.Setenv("DRYCC_REDIS_STREAM_GROUP", original.RedisStreamGroup)
	os.Setenv("AGGREGATOR_STOP_TIMEOUT_SEC", fmt.Sprint(original.StopTimeoutSeconds))
}
