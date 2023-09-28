#!/usr/bin/env bash

DEV_ENV_OPTS=$2
DEV_ENV_IMAGE=$3

function start-test-redis() {
  podman run --name test-logger-redis -d docker.io/library/redis:latest
}

function stop-test-redis() {
  podman kill test-logger-redis
  podman rm test-logger-redis
}

function test-cover() {
  start-test-redis
  REDIS_IP=$(podman inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-redis)
  podman run ${DEV_ENV_OPTS} \
    -e DRYCC_REDIS_ADDRS=${REDIS_IP}:6379 \
    -it \
    ${DEV_ENV_IMAGE} \
    bash -c "test-cover.sh"
  stop-test-redis
}

function test-unit() {
  start-test-redis test --cover --race -v
  REDIS_IP=$(podman inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-redis)
  echo "redis ip: $REDIS_IP"
  podman run ${DEV_ENV_OPTS} \
    -e DRYCC_REDIS_ADDRS=${REDIS_IP}:6379 \
    -it \
    ${DEV_ENV_IMAGE} \
    bash -c "go test --cover --race -v -tags=testredis ./..."
  stop-test-redis
}

"$@"
