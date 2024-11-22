package log

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	appName = "logger"
)

type config struct {
	ValkeyURL          string `envconfig:"DRYCC_VALKEY_URL" default:"redis://127.0.0.1:6379"`
	ValkeyStream       string `envconfig:"DRYCC_VALKEY_STREAM" default:"logs"`
	ValkeyStreamGroup  string `envconfig:"DRYCC_VALKEY_STREAM_GROUP" default:"logger"`
	StopTimeoutSeconds int    `envconfig:"AGGREGATOR_STOP_TIMEOUT_SEC" default:"1"`
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
