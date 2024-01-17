package storage

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	r "github.com/redis/go-redis/v9"
)

type message struct {
	app         string
	messageBody string
}

func newMessage(app string, messageBody string) *message {
	return &message{
		app:         app,
		messageBody: messageBody,
	}
}

type messagePipeliner struct {
	timeout       time.Duration
	bufferSize    int
	messageCount  int
	pipeline      r.Pipeliner
	timeoutTicker *time.Ticker
	queuedApps    map[string]bool
	errCh         chan error
}

func newMessagePipeliner(bufferSize int, redisClient *r.ClusterClient, timeout time.Duration, errCh chan error) *messagePipeliner {
	return &messagePipeliner{
		timeout:       timeout,
		bufferSize:    bufferSize,
		pipeline:      redisClient.Pipeline(),
		timeoutTicker: time.NewTicker(timeout),
		queuedApps:    map[string]bool{},
		errCh:         errCh,
	}
}

func (mp *messagePipeliner) addMessage(message *message) {
	ctx, cancel := context.WithTimeout(context.Background(), mp.timeout)
	defer cancel()
	if err := mp.pipeline.Publish(ctx, message.app, message.messageBody).Err(); err != nil {
		mp.errCh <- fmt.Errorf("error adding publish to %s to the pipeline: %s", message.app, err)
	} else if err := mp.pipeline.RPush(ctx, message.app, message.messageBody).Err(); err != nil {
		mp.errCh <- fmt.Errorf("error adding rpush to %s to the pipeline: %s", message.app, err)
	} else {
		mp.queuedApps[message.app] = true
		mp.messageCount++
	}
}

func (mp messagePipeliner) execPipeline() {
	ctx, cancel := context.WithTimeout(context.Background(), mp.timeout)
	defer cancel()
	for app := range mp.queuedApps {
		if err := mp.pipeline.LTrim(ctx, app, int64(-1*mp.bufferSize), -1).Err(); err != nil {
			log.Printf("error adding ltrim of %s to the pipeline: %s", app, err)
			mp.errCh <- fmt.Errorf("error adding ltrim of %s to the pipeline: %s", app, err)
		}
	}
	if _, err := mp.pipeline.Exec(ctx); err != nil {
		log.Printf("error executing pipeline: %s", err)
		mp.errCh <- fmt.Errorf("error executing pipeline: %s", err)
	}
}

func newClusterClient(cfg *redisConfig) (*r.ClusterClient, error) {
	addrs := strings.Split(cfg.Addrs, ",")
	sort.Strings(addrs)
	return r.NewClusterClient(&r.ClusterOptions{
		ClusterSlots: func(context.Context) ([]r.ClusterSlot, error) {
			const slotsSize = 16383
			var size = len(addrs)
			var slotsRange = slotsSize / size
			var slots []r.ClusterSlot
			for index, addr := range addrs {
				start := slotsRange * index
				end := start + slotsRange
				if (slotsSize - end) < slotsRange {
					end = slotsSize
				}
				slots = append(slots, r.ClusterSlot{
					Start: start,
					End:   end,
					Nodes: []r.ClusterNode{{Addr: addr}},
				})
			}
			return slots, nil
		},
		Password:      cfg.Password, // "" == no password
		RouteRandomly: true,
	}), nil
}

type redisAdapter struct {
	started        bool
	bufferSize     int
	redisClient    *r.ClusterClient
	messageChannel chan *message
	stopCh         chan struct{}
	config         *redisConfig
}

// NewRedisStorageAdapter returns a pointer to a new instance of a redis-based storage.Adapter.
func NewRedisStorageAdapter(bufferSize int) (Adapter, error) {
	if bufferSize <= 0 {
		return nil, fmt.Errorf("invalid buffer size: %d", bufferSize)
	}
	cfg, err := parseConfig(appName)
	if err != nil {
		return nil, err
	}
	client, err := newClusterClient(cfg)
	if err != nil {
		return nil, err
	}
	rsa := &redisAdapter{
		bufferSize:     bufferSize,
		redisClient:    client,
		messageChannel: make(chan *message, bufferSize),
		stopCh:         make(chan struct{}),
		config:         cfg,
	}
	return rsa, nil
}

// Start the storage adapter. Invocations of this function are not concurrency safe and multiple
// serialized invocations have no effect.
func (a *redisAdapter) Start() {
	if !a.started {
		a.started = true
		mp := newMessagePipeliner(a.bufferSize, a.redisClient, a.config.PipelineTimeout, make(chan error, a.bufferSize))
		go func() {
			for {
				select {
				case err := <-mp.errCh:
					log.Printf("select pipeline message err: %v", err)
				case <-a.stopCh:
					return
				case message := <-a.messageChannel:
					mp.addMessage(message)
					if mp.messageCount == a.config.PipelineLength {
						mp.execPipeline()
					}
				case <-mp.timeoutTicker.C:
					mp.execPipeline()
				}
			}
		}()
	}
}

// Write adds a log message to to an app-specific list in redis using ring-buffer-like semantics
func (a *redisAdapter) Write(app string, messageBody string) error {
	a.messageChannel <- newMessage(app, messageBody)
	return nil
}

// Read retrieves a specified number of log lines from an app-specific list in redis
func (a *redisAdapter) Read(app string, lines int) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.config.PipelineTimeout)
	defer cancel()
	stringSliceCmd := a.redisClient.LRange(ctx, app, int64(-1*lines), -1)
	result, err := stringSliceCmd.Result()
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}
	return nil, fmt.Errorf("could not find logs for '%s'", app)
}

// Make Chan a pipeline to read logs all the time
func (a *redisAdapter) Chan(ctx context.Context, app string, size int) (chan string, error) {
	channel := make(chan string, size)
	go func() {
		defer close(channel)
		subscribe := a.redisClient.Subscribe(context.Background(), app)
		defer subscribe.Close()
		subscriptions := subscribe.Channel()
		for len(channel) != size {
			select {
			case <-ctx.Done():
				return
			case message := <-subscriptions:
				channel <- message.Payload
			}
		}
	}()
	return channel, nil
}

// Destroy deletes an app-specific list from redis
func (a *redisAdapter) Destroy(app string) error {
	ctx, cancel := context.WithTimeout(context.Background(), a.config.PipelineTimeout)
	defer cancel()
	return a.redisClient.Del(ctx, app).Err()
}

// Reopen the storage adapter-- in the case of this implementation, a no-op
func (a *redisAdapter) Reopen() error {
	return nil
}

// Stop the storage adapter. Additional writes may not be performed after stopping.
func (a *redisAdapter) Stop() {
	close(a.stopCh)
}
