package log

import (
	"context"
	l "log"
	"strings"
	"testing"
	"time"

	"github.com/drycc/logger/storage"
	valkey "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestAggregator(t *testing.T) {
	storageAdapter, err := storage.NewAdapter("file", 100)
	assert.NoError(t, err)
	aggregator, err := NewAggregator("valkey", storageAdapter)
	assert.NoError(t, err)
	err = aggregator.Listen()
	assert.NoError(t, err)
	stoppedCh := aggregator.Stopped()
	err = aggregator.Stop()
	assert.NoError(t, err)
	stopErr := <-stoppedCh
	assert.NoError(t, stopErr, "aggregator stopped with error")
}

func generateTestData(ctx context.Context, count int, message map[string]interface{}) {
	cfg, err := parseConfig(appName)
	if err != nil {
		l.Fatalf("config error: %s: ", err)
	}
	valkeyAddrs := strings.Split(cfg.ValkeyAddrs, ",")
	valkeyClient := valkey.NewClient(&valkey.Options{
		Addr:     valkeyAddrs[0],
		Password: cfg.ValkeyPassword,
	})
	if err != nil {
		l.Println(err)
	}
	for i := 0; i < count; i++ {
		valkeyClient.XAdd(ctx, &valkey.XAddArgs{
			Stream: cfg.ValkeyStream,
			ID:     "*",
			Values: message,
		})
	}
}

func TestAggregatorMessageMainLoop(t *testing.T) {
	message := map[string]interface{}{
		"data": "Hello world",
	}
	messageCount := 10
	ctx, cancel := context.WithCancel(context.Background())
	generateTestData(ctx, messageCount, message)
	msg := make(chan map[string]interface{}, messageCount)
	aggregator := valkeyAggregator{
		handle: func(message map[string]interface{}) {
			msg <- message
		},
		ctx:    ctx,
		cancel: cancel,
	}
	aggregator.Listen()
	for i := 0; i < messageCount; i++ {
		select {
		case expect := <-msg:
			assert.Equal(t, message, expect)
		case <-time.After(time.Second * 10):
			t.Error("messageMainLoop timeout")
		}
	}
	err := aggregator.Stop()
	assert.NoError(t, err)
}
