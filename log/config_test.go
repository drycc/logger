package log

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNsqURL(t *testing.T) {
	c := config{
		NSQAddresses: "somehost:3333",
	}
	assert.Equal(t, c.nsqURLs(), []string{"somehost:3333"})
}

func TestStopTimeoutDuration(t *testing.T) {
	c := config{
		StopTimeoutSeconds: 60,
	}
	assert.Equal(t, c.stopTimeoutDuration(), time.Duration(c.StopTimeoutSeconds)*time.Second)
}

func TestParseConfig(t *testing.T) {
	os.Setenv("NSQ_TOPIC", "topic")
	os.Setenv("NSQ_CHANNEL", "channel")
	os.Setenv("NSQ_HANDLER_COUNT", "3")
	os.Setenv("AGGREGATOR_STOP_TIMEOUT_SEC", "2")

	addresses := os.Getenv("DRYCC_NSQD_ADDRS")

	c, err := parseConfig("foo")
	assert.NoError(t, err)
	assert.Equal(t, c.NSQAddresses, addresses)
	assert.Equal(t, c.NSQTopic, "topic")
	assert.Equal(t, c.NSQChannel, "channel")
	assert.Equal(t, c.NSQHandlerCount, 3)
	assert.Equal(t, c.StopTimeoutSeconds, 2)
}
