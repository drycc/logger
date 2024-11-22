package storage

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestValkeyReadFromNonExistingApp(t *testing.T) {
	// Initialize a new storage adapter
	a, err := NewValkeyStorageAdapter(10)
	if err != nil {
		t.Error(err)
	}
	// No logs have been written; there should be no valkey list for app
	messages, err := a.Read(app, 10)
	if messages != nil {
		t.Error("expected no messages, but got some")
	}
	if err == nil || err.Error() != fmt.Sprintf("could not find logs for '%s'", app) {
		t.Error("did not receive expected error message")
	}
}

func TestValkeyWithBadBufferSizes(t *testing.T) {
	// Initialize with invalid buffer sizes
	for _, size := range []int{-1, 0} {
		a, err := NewValkeyStorageAdapter(size)
		if a != nil {
			t.Error("expected no storage adapter, but got one")
		}
		if err == nil || err.Error() != fmt.Sprintf("invalid buffer size: %d", size) {
			t.Error("did not receive expected error message")
		}
	}
}

func TestValkeyLogs(t *testing.T) {
	// Initialize with small buffers
	a, err := NewValkeyStorageAdapter(10)
	if err != nil {
		t.Error(err)
	}
	a.Start()
	defer a.Stop()
	// And write a few logs to it, but do NOT fill it up
	for i := 0; i < 5; i++ {
		if err := a.Write(app, fmt.Sprintf("message %d", i)); err != nil {
			t.Error(err)
		}
	}
	// Sleep for a bit because the adapter queues logs internally and writes them to Valkey only when
	// there are 50 queued up OR a 1 second timeout has been reached.
	time.Sleep(time.Second * 2)
	// Read more logs than there are
	messages, err := a.Read(app, 8)
	if err != nil {
		t.Error(err)
	}
	// Should only get as many messages as we actually have
	if len(messages) != 5 {
		t.Errorf("only expected 5 log messages, got %d", len(messages))
	}
	// Read fewer logs than there are
	messages, err = a.Read(app, 3)
	if err != nil {
		t.Error(err)
	}
	// Should get the 3 MOST RECENT logs
	if len(messages) != 3 {
		t.Errorf("only expected 5 log messages, got %d", len(messages))
	}
	for i := 0; i < 3; i++ {
		expectedMessage := fmt.Sprintf("message %d", i+2)
		if messages[i] != expectedMessage {
			t.Errorf("expected: \"%s\", got \"%s\"", expectedMessage, messages[i])
		}
	}
	// Overfill the buffer
	for i := 5; i < 11; i++ {
		if err := a.Write(app, fmt.Sprintf("message %d", i)); err != nil {
			t.Error(err)
		}
	}
	// Sleep for a bit because the adapter queues logs internally and writes them to Valkey only when
	// there are 50 queued up OR a 1 second timeout has been reached.
	time.Sleep(time.Second * 2)
	// Read more logs than the buffer can hold
	messages, err = a.Read(app, 20)
	if err != nil {
		t.Error(err)
	}
	// Should only get as many messages as the buffer can hold
	if len(messages) != 11 {
		t.Errorf("only expected 10 log messages, got %d", len(messages))
	}
	// And they should only be the 10 MOST RECENT logs
	for i := 0; i < 11; i++ {
		expectedMessage := fmt.Sprintf("message %d", i)
		if messages[i] != expectedMessage {
			t.Errorf("expected: \"%s\", got \"%s\"", expectedMessage, messages[i])
		}
	}
}

func TestValkeyDestroy(t *testing.T) {
	a, err := NewValkeyStorageAdapter(10)
	if err != nil {
		t.Error(err)
	}
	a.Start()
	defer a.Stop()
	// Write a log to create the file
	if err := a.Write(app, "Hello, log!"); err != nil {
		t.Error(err)
	}
	// Sleep for a bit because the adapter queues logs internally and writes them to Valkey only when
	// there are 50 queued up OR a 1 second timeout has been reached.
	time.Sleep(time.Second * 2)
	var ctx = context.Background()
	// A valkey list should exist for the app
	adapter := a.(*valkeyAdapter).valkeyClient
	exists, err := adapter.Exists(ctx, app).Result()
	if err != nil {
		t.Error(err)
	}
	if !(exists == 1) {
		t.Error("Log valkey list was expected to exist, but doesn't.")
	}
	// Now destroy it
	if err := a.Destroy(app); err != nil {
		t.Error(err)
	}
	// Now check that the valkey list no longer exists
	exists, err = adapter.Exists(ctx, app).Result()
	if err != nil {
		t.Error(err)
	}
	if exists == 1 {
		t.Error("Log valkey list still exist, but was expected not to.")
	}
}

func TestValkeyChan(t *testing.T) {
	// Write a log to create the file
	adapter, err := NewValkeyStorageAdapter(100)

	if err != nil {
		t.Error(err)
	}
	adapter.Start()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	channel, err := adapter.Chan(ctx, app, 100)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	for i := 0; i < 10; i++ {
		if err := adapter.Write(app, fmt.Sprintf("Hello, log %d !", i)); err != nil {
			t.Error(err)
		}
	}

	for i := 0; i < 10; i++ {
		line := <-channel
		expected := fmt.Sprintf("Hello, log %d !", i)
		if line != expected {
			t.Error("the log content does not match the expectation.", expected, line)
		}
	}

	if line := <-channel; line != "" {
		t.Error("expected timeout returned null, but found: ", line)
	}
}
