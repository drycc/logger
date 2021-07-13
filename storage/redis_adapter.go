package storage

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	r "github.com/go-redis/redis/v8"
)

var ctx = context.Background()

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
	bufferSize    int
	messageCount  int
	pipeline      r.Pipeliner
	timeoutTicker *time.Ticker
	queuedApps    map[string]bool
	errCh         chan error
}

func newMessagePipeliner(bufferSize int, redisClient *r.ClusterClient, timeout time.Duration, errCh chan error) *messagePipeliner {
	return &messagePipeliner{
		bufferSize:    bufferSize,
		pipeline:      redisClient.Pipeline(),
		timeoutTicker: time.NewTicker(timeout),
		queuedApps:    map[string]bool{},
		errCh:         errCh,
	}
}

func (mp *messagePipeliner) addMessage(message *message) {
	if err := mp.pipeline.RPush(ctx, message.app, message.messageBody).Err(); err == nil {
		mp.queuedApps[message.app] = true
		mp.messageCount++
	} else {
		mp.errCh <- fmt.Errorf("Error adding rpush to %s to the pipeline: %s", message.app, err)
	}
}

func (mp messagePipeliner) execPipeline() {
	for app := range mp.queuedApps {
		if err := mp.pipeline.LTrim(ctx, app, int64(-1*mp.bufferSize), -1).Err(); err != nil {
			mp.errCh <- fmt.Errorf("Error adding ltrim of %s to the pipeline: %s", app, err)
		}
	}
	if _, err := mp.pipeline.Exec(ctx); err != nil {
		mp.errCh <- fmt.Errorf("Error executing pipeline: %s", err)
	}
}

func newRedisClusterSlots(addrs []string) func() ([]r.ClusterSlot, error) {
	return func() ([]r.ClusterSlot, error) {
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
	}
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
		return nil, fmt.Errorf("Invalid buffer size: %d", bufferSize)
	}
	cfg, err := parseConfig(appName)
	if err != nil {
		log.Fatalf("config error: %s: ", err)
	}
	if err != nil {
		return nil, err
	}
	addrs := strings.Split(cfg.Addrs, ",")
	sort.Strings(addrs)
	rsa := &redisAdapter{
		bufferSize: bufferSize,
		redisClient: r.NewClusterClient(&r.ClusterOptions{
			ClusterSlots:  newRedisClusterSlots(addrs),
			Password:      cfg.Password, // "" == no password
			RouteRandomly: true,
		}),
		messageChannel: make(chan *message),
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
		errCh := make(chan error)
		mp := newMessagePipeliner(a.bufferSize, a.redisClient, a.config.PipelineTimeout, errCh)
		go func() {
			defer mp.pipeline.Close()
			for {
				select {
				case err := <-errCh:
					log.Println(err)
				case <-a.stopCh:
					return
				case message := <-a.messageChannel:
					mp.addMessage(message)
					if mp.messageCount == a.config.PipelineLength {
						go mp.execPipeline()
					}
				case <-mp.timeoutTicker.C:
					go mp.execPipeline()
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
	stringSliceCmd := a.redisClient.LRange(ctx, app, int64(-1*lines), -1)
	result, err := stringSliceCmd.Result()
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}
	return nil, fmt.Errorf("Could not find logs for '%s'", app)
}

// Destroy deletes an app-specific list from redis
func (a *redisAdapter) Destroy(app string) error {
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
