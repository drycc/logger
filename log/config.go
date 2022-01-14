package log

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	appName = "logger"
)

type config struct {
	RedisAddrs         string `envconfig:"DRYCC_REDIS_ADDRS" default:":6379"`
	RedisPassword      string `envconfig:"DRYCC_REDIS_PASSWORD" default:""`
	RedisStream        string `envconfig:"DRYCC_REDIS_STREAM" default:"logs"`
	RedisStreamGroup   string `envconfig:"DRYCC_REDIS_STREAM_GROUP" default:"logger"`
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
