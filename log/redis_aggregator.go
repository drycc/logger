package log

import (
	"context"
	l "log"
	"strings"
	"time"

	"github.com/drycc/logger/storage"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type redisAggregator struct {
	listening bool
	cfg       *config
	ctx       context.Context
	handle    func(map[string]interface{})
	cancel    context.CancelFunc
}

func newRedisAggregator(storageAdapter storage.Adapter) Aggregator {
	context, cancel := context.WithCancel(context.Background())
	return &redisAggregator{
		handle: func(message map[string]interface{}) {
			err := handle([]byte(message["data"].(string)), storageAdapter)
			if err != nil {
				l.Printf("handle message error: %v, %v", err, message)
			}
		},
		ctx:    context,
		cancel: cancel,
	}
}

func (a *redisAggregator) messageMainLoop(redisAddr string) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: a.cfg.RedisPassword,
	})
	redisClient.XGroupCreateMkStream(a.ctx, a.cfg.RedisStream, a.cfg.RedisStreamGroup, "0")
	xReadGroupArgs := redis.XReadGroupArgs{
		Group:    a.cfg.RedisStreamGroup,
		Consumer: uuid.New().String(),
		Streams:  []string{a.cfg.RedisStream, ">"},
		Count:    30,
		Block:    time.Duration(30) * time.Second,
		NoAck:    false,
	}
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			entries, err := redisClient.XReadGroup(a.ctx, &xReadGroupArgs).Result()
			if err == nil && len(entries) > 0 {
				for i := 0; i < len(entries[0].Messages); i++ {
					a.handle(entries[0].Messages[i].Values)
					redisClient.XAck(a.ctx, a.cfg.RedisStream, a.cfg.RedisStreamGroup, entries[0].Messages[i].ID)
				}
			} else {
				l.Printf("no data was read from redis xread group, %v, %v", err, entries)
				time.Sleep(time.Duration(9) * time.Second)
			}
		}
	}
}

// Listen starts the aggregator. Invocations of this function are not concurrency safe and multiple
// serialized invocations have no effect.
func (a *redisAggregator) Listen() error {
	// Should only ever be called once
	if !a.listening {
		a.listening = true
		var err error
		a.cfg, err = parseConfig(appName)
		if err != nil {
			l.Fatalf("config error: %s: ", err)
		}
		redisAddrs := strings.Split(a.cfg.RedisAddrs, ",")
		for _, redisAddr := range redisAddrs {
			go a.messageMainLoop(redisAddr)
		}
	}
	return nil
}

// Stop is the Aggregator interface implementation
func (a *redisAggregator) Stop() error {
	a.cancel()
	timeout := a.cfg.stopTimeoutDuration()
	tmr := time.NewTimer(timeout)
	defer tmr.Stop()
	select {
	case <-tmr.C:
		return newErrStopTimedOut(timeout)
	case <-a.ctx.Done():
		return nil
	}
}

// Stopped is the Aggregator interface implementation
func (a *redisAggregator) Stopped() <-chan error {
	retCh := make(chan error)
	go func() {
		<-a.ctx.Done()
		retCh <- nil
	}()
	return retCh
}
