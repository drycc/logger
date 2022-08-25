package storage

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func TestReadFromNonExistingApp(t *testing.T) {
	var err error
	LogRoot, err = ioutil.TempDir("", "log-tests")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(LogRoot)
	// Initialize a new storage adapter
	a, err := NewFileAdapter()
	if err != nil {
		t.Error(err)
	}
	// No logs have been writter; there should be no ringBuffer for app
	messages, err := a.Read(app, 10)
	if messages != nil {
		t.Error("Expected no messages, but got some")
	}
	if err == nil || err.Error() != fmt.Sprintf("Could not find logs for '%s'", app) {
		t.Error("Did not receive expected error message")
	}
}

func TestLogs(t *testing.T) {
	var err error
	LogRoot, err = ioutil.TempDir("", "log-tests")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(LogRoot)
	a, err := NewFileAdapter()
	if err != nil {
		t.Error(err)
	}
	// And write a few logs
	for i := 0; i < 5; i++ {
		if err := a.Write(app, fmt.Sprintf("message %d", i)); err != nil {
			t.Error(err)
		}
	}
	// Read more logs than there are
	messages, err := a.Read(app, 8)
	if err != nil {
		t.Error(err)
	}
	// Should only get as many messages as we actually have
	if len(messages) != 5 {
		t.Error("only expected 5 log messages")
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
}

func TestFileChan(t *testing.T) {
	var err error
	LogRoot, err = ioutil.TempDir("", "log-tests")
	LogRoot = "/tmp"
	if err != nil {
		t.Error(err)
	}
	//defer os.Remove(LogRoot)
	sa, err := NewFileAdapter()
	if err != nil {
		t.Error(err)
	}
	if err := sa.Write(app, "Hello, log!"); err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	channel, err := sa.Chan(ctx, app, 100)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 1)
	// Write a log to create the file
	for i := 0; i < 10; i++ {
		if err := sa.Write(app, fmt.Sprintf("Hello, log %d !", i)); err != nil {
			t.Error(err)
		}
	}
	for i := 0; i < 10; i++ {
		line := <-channel
		expected := fmt.Sprintf("Hello, log %d !", i)
		if line != expected {
			t.Errorf("expected: %s\nactual: %s\n", expected, line)
		}
	}
	if line := <-channel; line != "" {
		t.Error("Expected timeout returned null, but found: ", line)
	}
}

func TestDestroy(t *testing.T) {
	var err error
	LogRoot, err = ioutil.TempDir("", "log-tests")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(LogRoot)
	sa, err := NewFileAdapter()
	if err != nil {
		t.Error(err)
	}

	a, ok := sa.(*fileAdapter)
	if !ok {
		t.Fatalf("returned adapter was not a ringBuffer")
	}

	// Write a log to create the file
	if err := a.Write(app, "Hello, log!"); err != nil {
		t.Error(err)
	}

	filename := path.Join(LogRoot, fmt.Sprintf("%s.log", app))
	// Test log file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("Log file was expected to exist, but doesn't.")
	}
	// Now destroy it
	if err := a.Destroy(app); err != nil {
		t.Error(err)
	}
	// Now check that the file no longer exists
	if _, err := os.Stat(filename); err == nil {
		t.Error("Log file still exists, but was expected not to.")
	}
	// And that we have no reference to it
	if _, ok := a.files[app]; ok {
		t.Error("Log fiel reference still exist, but was expected not to.")
	}
}

func TestReopen(t *testing.T) {
	var err error
	LogRoot, err = ioutil.TempDir("", "log-tests")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(LogRoot)
	sa, err := NewFileAdapter()
	if err != nil {
		t.Error(err)
	}
	a, ok := sa.(*fileAdapter)
	if !ok {
		t.Fatalf("returned adapter was not a ringBuffer")
	}
	// Write a log to create the file
	if err := a.Write(app, "Hello, log!"); err != nil {
		t.Error(err)
	}
	// At least one file reference should exist
	if len(a.files) == 0 {
		t.Error("At least one log file reference expected to exist, but doesn't.")
	}
	// Now "reopen" logs
	if err := a.Reopen(); err != nil {
		t.Error(err)
	}
	// Now check that no file references exist
	if len(a.files) != 0 {
		t.Error("At least one log file reference still exists, but was expected not to.")
	}
}
