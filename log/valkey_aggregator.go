package log

import (
	"context"
	l "log"
	"time"

	"github.com/drycc/logger/storage"
	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/valkeycompat"
)

type valkeyAggregator struct {
	listening bool
	cfg       *config
	ctx       context.Context
	handle    func(map[string]interface{})
	cancel    context.CancelFunc
}

func newValkeyAggregator(storageAdapter storage.Adapter) Aggregator {
	context, cancel := context.WithCancel(context.Background())
	return &valkeyAggregator{
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

func (a *valkeyAggregator) messageMainLoop() {
	valkeyClient, _ := valkey.NewClient(valkey.MustParseURL(a.cfg.ValkeyURL))
	valkeyCmdable := valkeycompat.NewAdapter(valkeyClient)
	valkeyCmdable.XGroupCreateMkStream(a.ctx, a.cfg.ValkeyStream, a.cfg.ValkeyStreamGroup, "0")

	xReadGroupArgs := valkeycompat.XReadGroupArgs{
		Group:    a.cfg.ValkeyStreamGroup,
		Consumer: uuid.New().String(),
		Streams:  []string{a.cfg.ValkeyStream, ">"},
		Count:    30,
		Block:    time.Duration(30) * time.Second,
		NoAck:    false,
	}
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			entries, err := valkeyCmdable.XReadGroup(a.ctx, xReadGroupArgs).Result()
			if err != nil {
				valkeyClient.Close()
				valkeyClient, _ = valkey.NewClient(valkey.MustParseURL(a.cfg.ValkeyURL))
				valkeyCmdable = valkeycompat.NewAdapter(valkeyClient)
			} else if len(entries) > 0 {
				for i := 0; i < len(entries[0].Messages); i++ {
					a.handle(entries[0].Messages[i].Values)
					valkeyCmdable.XAck(a.ctx, a.cfg.ValkeyStream, a.cfg.ValkeyStreamGroup, entries[0].Messages[i].ID)
				}
			} else {
				l.Printf("no data was read from valkey xread group, %v, %v", err, entries)
				time.Sleep(time.Duration(9) * time.Second)
			}
		}
	}
}

// Listen starts the aggregator. Invocations of this function are not concurrency safe and multiple
// serialized invocations have no effect.
func (a *valkeyAggregator) Listen() error {
	// Should only ever be called once
	if !a.listening {
		a.listening = true
		var err error
		a.cfg, err = parseConfig(appName)
		if err != nil {
			l.Fatalf("config error: %s: ", err)
		}
		go a.messageMainLoop()
	}
	return nil
}

// Stop is the Aggregator interface implementation
func (a *valkeyAggregator) Stop() error {
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
func (a *valkeyAggregator) Stopped() <-chan error {
	retCh := make(chan error)
	go func() {
		<-a.ctx.Done()
		retCh <- nil
	}()
	return retCh
}
