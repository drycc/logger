package storage

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	appName = "logger"
)

type valkeyConfig struct {
	URL                    string `envconfig:"DRYCC_VALKEY_URL" default:"redis://127.0.0.1:6379"`
	PipelineLength         int    `envconfig:"DRYCC_VALKEY_PIPELINE_LENGTH" default:"50"`
	PipelineTimeoutSeconds int    `envconfig:"DRYCC_VALKEY_PIPELINE_TIMEOUT_SECONDS" default:"30"`
	PipelineTimeout        time.Duration
}

func parseConfig(appName string) (*valkeyConfig, error) {
	ret := new(valkeyConfig)
	if err := envconfig.Process(appName, ret); err != nil {
		return nil, err
	}
	ret.PipelineTimeout = time.Duration(ret.PipelineTimeoutSeconds) * time.Second
	return ret, nil
}
