package log

import (
	"encoding/json"
	"testing"

	"github.com/drycc/logger/storage"
	"github.com/stretchr/testify/assert"
)

var (
	validControllerMessage = `{"log": "INFO [foo]: admin deployed 2fd9226", "time": "2016-10-18T20:29:38+00:00", "stream": "stderr", "kubernetes": {"namespace_name": "foo", "pod_id": "podId", "pod_name": "podName", "container_name": "drycc-controller", "labels": {"app": "foo",
"heritage": "drycc", "type": "web", "version": "v2"}, "host": "host"}}`

	invalidAppMessage = `{"log": "test message", "stream": "stderr", "time": "2016-10-18T20:29:38+00:00", "kubernetes": {"namespace_name": "foo", "pod_id": "podId", "pod_name": "foo-web-845861952-nzf60", "container_name": "foo-web", "labels": {"app": "foo",
"heritage": "drycc", "type": "web", "version": "v2"}, "host": "host"}}`

	badjson = `{"log":}`
)

func TestValidControllerMessage(t *testing.T) {
	message := new(Message)
	err := json.Unmarshal([]byte(validControllerMessage), message)
	assert.NoError(t, err, "error occurred parsing log message")
	assert.True(t, fromController(message), "json is not from controller")
}

func TestInvalidControllerMessage(t *testing.T) {
	message := new(Message)
	err := json.Unmarshal([]byte(invalidAppMessage), message)
	assert.NoError(t, err, "error occurred parsing log message")
	assert.False(t, fromController(message), "valid controller message")
}

func TestGetApplicationFromValidControllerMessage(t *testing.T) {
	message := new(Message)
	err := json.Unmarshal([]byte(validControllerMessage), message)
	assert.NoError(t, err, "error occurred parsing log message")
	expected := getApplicationFromControllerMessage(message)
	assert.Equal(t, expected, "foo", "failed to retrieve app from message")
}

func TestBuildControllerLogMessageFromValidMessage(t *testing.T) {
	message := new(Message)
	err := json.Unmarshal([]byte(validControllerMessage), message)
	assert.NoError(t, err, "error occurred parsing log message")
	expected := buildControllerLogMessage(message)
	assert.Equal(t, expected,
		"2016-10-18T20:29:38+00:00 drycc[controller]: INFO admin deployed 2fd9226",
		"failed to build controller log")
}

func TestHandleValidControllerMessage(t *testing.T) {
	storage.LogRoot = "/tmp"
	a, err := storage.NewAdapter("file", 1)
	assert.NoError(t, err, "error creating ring buffer")
	err = handle([]byte(validControllerMessage), a)
	assert.NoError(t, err, "error occurred storing log message")
	expected, _ := a.Read("foo", 1)
	assert.Equal(t, expected[0],
		"2016-10-18T20:29:38+00:00 drycc[controller]: INFO admin deployed 2fd9226",
		"failed to acquire controller log message")
}

func TestHandleInvalidControllerMessage(t *testing.T) {
	storage.LogRoot = "/tmp"
	a, err := storage.NewAdapter("file", 1)
	assert.NoError(t, err, "error creating ring buffer")
	err = handle([]byte(badjson), a)
	assert.Error(t, err, "no error occurred parsing json")
}
