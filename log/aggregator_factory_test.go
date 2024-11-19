package log

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

type stubStorageAdapter struct {
}

func (a *stubStorageAdapter) Start() {
}

func (a *stubStorageAdapter) Write(string, string) error {
	return nil
}

func (a *stubStorageAdapter) Read(string, int) ([]string, error) {
	return []string{}, nil
}

func (a *stubStorageAdapter) Chan(context.Context, string, int) (chan string, error) {
	return make(chan string), nil
}

func (a *stubStorageAdapter) Destroy(string) error {
	return nil
}

func (a *stubStorageAdapter) Reopen() error {
	return nil
}

func (a *stubStorageAdapter) Stop() {
}

func TestGetUsingInvalidValues(t *testing.T) {
	_, err := NewAggregator("bogus", &stubStorageAdapter{})
	if err == nil || err.Error() != fmt.Sprintf("unrecognized aggregator type: '%s'", "bogus") {
		t.Error("did not receive expected error message")
	}
}

func TestValkeyBasedAggregator(t *testing.T) {
	a, err := NewAggregator("valkey", &stubStorageAdapter{})
	if err != nil {
		t.Error(err)
	}
	expected := "*log.valkeyAggregator"
	aType := reflect.TypeOf(a).String()
	if aType != expected {
		t.Errorf("Expected a %s, but got a %s", expected, aType)
	}
}
