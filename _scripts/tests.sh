#!/usr/bin/env bash

DEV_ENV_OPTS=$2
DEV_ENV_IMAGE=$3

function start-test-valkey() {
  podman run --name test-logger-valkey -d docker.io/valkey/valkey:latest
}

function stop-test-valkey() {
  podman kill test-logger-valkey
  podman rm test-logger-valkey
}

function test-cover() {
  start-test-valkey
  VALKEY_IP=$(podman inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-valkey)
  podman run ${DEV_ENV_OPTS} \
    -e DRYCC_VALKEY_URL=redis://${VALKEY_IP}:6379/0 \
    -it \
    ${DEV_ENV_IMAGE} \
    bash -c "test-cover.sh"
  stop-test-valkey
}

function test-unit() {
  start-test-valkey test --cover --race -v
  VALKEY_IP=$(podman inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-valkey)
  echo "valkey ip: $VALKEY_IP"
  podman run ${DEV_ENV_OPTS} \
    -e DRYCC_VALKEY_URL=redis://${VALKEY_IP}:6379/0 \
    -e DRYCC_VALKEY_PIPELINE_LENGTH=1 \
    -e DRYCC_VALKEY_PIPELINE_TIMEOUT_SECONDS=1 \
    -it \
    ${DEV_ENV_IMAGE} \
    bash -c "go test --cover --race -v -tags=testvalkey ./..."
  stop-test-valkey
}

"$@"
