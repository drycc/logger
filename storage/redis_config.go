package storage

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	appName = "logger"
)

type redisConfig struct {
	Addrs                  string `envconfig:"DRYCC_REDIS_ADDRS" default:":6379"`
	Password               string `envconfig:"DRYCC_REDIS_PASSWORD" default:""`
	PipelineLength         int    `envconfig:"DRYCC_REDIS_PIPELINE_LENGTH" default:"50"`
	PipelineTimeoutSeconds int    `envconfig:"DRYCC_REDIS_PIPELINE_TIMEOUT_SECONDS" default:"30"`
	PipelineTimeout        time.Duration
}

func parseConfig(appName string) (*redisConfig, error) {
	ret := new(redisConfig)
	if err := envconfig.Process(appName, ret); err != nil {
		return nil, err
	}
	ret.PipelineTimeout = time.Duration(ret.PipelineTimeoutSeconds) * time.Second
	return ret, nil
}
