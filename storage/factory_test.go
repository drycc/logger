package storage

import (
	"reflect"
	"testing"
)

const (
	app = "test-app"
)

func TestFactoryGetUsingInvalidValues(t *testing.T) {
	const adapterType = "bogus"
	_, err := NewAdapter(adapterType, 1)
	if err == nil {
		t.Fatalf("did not receive an error message")
	}
	unrecognizedErr, ok := err.(errUnrecognizedStorageAdapterType)
	if !ok {
		t.Fatalf("expected an errUnrecognizedStorageAdapterType, received %s", err)
	}
	if unrecognizedErr.adapterType != adapterType {
		t.Fatalf("got an errUnrecognizedStorageAdapterType, but expected adapter type %s, got %s", adapterType, unrecognizedErr.adapterType)
	}
}

func TestFactoryGetFileBasedAdapter(t *testing.T) {
	a, err := NewAdapter("file", 1)
	if err != nil {
		t.Error(err)
	}
	retType, ok := a.(*fileAdapter)
	if !ok {
		t.Fatalf("expected a *fileAdapter, got %s", reflect.TypeOf(retType).String())
	}
}

func TestGetValkeyBasedAdapter(t *testing.T) {
	a, err := NewAdapter("valkey", 1)
	if err != nil {
		t.Error(err)
	}
	retType, ok := a.(*valkeyAdapter)
	if !ok {
		t.Errorf("expected a valkeyAdapter, but got a %s", reflect.TypeOf(retType).String())
	}
}
