package storage

import (
	"context"
	"fmt"
	"time"

	"container/list"

	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/valkeycompat"
)

type message struct {
	app         string
	messageBody string
}

type valkeyAdapter struct {
	started        bool
	bufferSize     int
	valkeyClient   valkeycompat.Cmdable
	messageChannel chan *message
	stopCh         chan struct{}
	config         *valkeyConfig
}

// NewValkeyStorageAdapter returns a pointer to a new instance of a valkey-based storage.Adapter.
func NewValkeyStorageAdapter(bufferSize int) (Adapter, error) {
	if bufferSize <= 0 {
		return nil, fmt.Errorf("invalid buffer size: %d", bufferSize)
	}
	cfg, err := parseConfig(appName)
	if err != nil {
		return nil, err
	}

	client, err := valkey.NewClient(valkey.MustParseURL(cfg.URL))

	if err != nil {
		return nil, err
	}
	rsa := &valkeyAdapter{
		bufferSize:     bufferSize,
		valkeyClient:   valkeycompat.NewAdapter(client),
		messageChannel: make(chan *message, bufferSize),
		stopCh:         make(chan struct{}),
		config:         cfg,
	}
	return rsa, nil
}

// Start the storage adapter. Invocations of this function are not concurrency safe and multiple
// serialized invocations have no effect.
func (a *valkeyAdapter) Start() {
	if !a.started {
		a.started = true
		ticker := time.NewTicker(a.config.PipelineTimeout)
		go func() {
			messages := list.New()
			for {
				select {
				case <-a.stopCh:
					a.execPublish(messages)
					return
				case message := <-a.messageChannel:
					messages.PushBack(message)
					if messages.Len() >= a.config.PipelineLength {
						a.execPublish(messages)
					}
				case <-ticker.C:
					a.execPublish(messages)
				}
			}
		}()
	}
}

func (a *valkeyAdapter) execPublish(messages *list.List) {
	ctx, cancel := context.WithTimeout(context.Background(), a.config.PipelineTimeout)
	defer cancel()

	a.valkeyClient.Pipelined(ctx, func(p valkeycompat.Pipeliner) error {
		for element := messages.Front(); element != nil; element = element.Next() {
			if message, ok := element.Value.(*message); ok {
				p.LTrim(ctx, message.app, int64(-1*a.bufferSize), -1)
				p.RPush(ctx, message.app, message.messageBody)
				p.Publish(ctx, message.app, message.messageBody)
			}
		}
		return nil
	})

	messages.Init()
}

// Write adds a log message to to an app-specific list in valkey using ring-buffer-like semantics
func (a *valkeyAdapter) Write(app string, messageBody string) error {
	a.messageChannel <- &message{
		app:         app,
		messageBody: messageBody,
	}
	return nil
}

// Read retrieves a specified number of log lines from an app-specific list in valkey
func (a *valkeyAdapter) Read(app string, lines int) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.config.PipelineTimeout)
	defer cancel()
	stringSliceCmd := a.valkeyClient.LRange(ctx, app, int64(-1*lines), -1)
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
func (a *valkeyAdapter) Chan(ctx context.Context, app string, size int) (chan string, error) {
	channel := make(chan string, size)

	go func() {
		defer close(channel)
		pubsub := a.valkeyClient.Subscribe(ctx, app)
		defer pubsub.Close()
		messages := pubsub.Channel()
		for len(channel) != size {
			select {
			case message := <-messages:
				channel <- message.Payload
			case <-ctx.Done():
				return
			}
		}
	}()

	return channel, nil
}

// Destroy deletes an app-specific list from valkey
func (a *valkeyAdapter) Destroy(app string) error {
	ctx, cancel := context.WithTimeout(context.Background(), a.config.PipelineTimeout)
	defer cancel()
	return a.valkeyClient.Del(ctx, app).Err()
}

// Reopen the storage adapter-- in the case of this implementation, a no-op
func (a *valkeyAdapter) Reopen() error {
	return nil
}

// Stop the storage adapter. Additional writes may not be performed after stopping.
func (a *valkeyAdapter) Stop() {
	close(a.stopCh)
}
