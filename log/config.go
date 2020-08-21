package log

import (
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	appName = "logger"
)

type config struct {
	NSQAddresses       string `envconfig:"DRYCC_NSQD_ADDRS" default:"127.0.0.1:4150"`
	NSQTopic           string `envconfig:"NSQ_TOPIC" default:"logs"`
	NSQChannel         string `envconfig:"NSQ_CHANNEL" default:"consume"`
	NSQHandlerCount    int    `envconfig:"NSQ_HANDLER_COUNT" default:"30"`
	StopTimeoutSeconds int    `envconfig:"AGGREGATOR_STOP_TIMEOUT_SEC" default:"1"`
}

func (c config) nsqURLs() []string {
	return strings.Split(c.NSQAddresses, ",")
}

func (c config) stopTimeoutDuration() time.Duration {
	return time.Duration(c.StopTimeoutSeconds) * time.Second
}

func parseConfig(appName string) (*config, error) {
	ret := new(config)
	if err := envconfig.Process(appName, ret); err != nil {
		return nil, err
	}
	return ret, nil
}
