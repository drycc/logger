#!/usr/bin/env bash

DEV_ENV_OPTS=$2
DEV_ENV_IMAGE=$3

function start-test-redis() {
  docker run --name test-logger-redis -d redis:latest
}

function start-test-nsqd() {
  docker run --name test-logger-nsqd -d nsqio/nsq nsqd
}

function stop-test-redis() {
  docker kill test-logger-redis
  docker rm test-logger-redis
}

function stop-test-nsqd() {
  docker kill test-logger-nsqd
  docker rm test-logger-nsqd
}

function test-cover() {
  start-test-redis
  start-test-nsqd
  REDIS_IP=$(docker inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-redis)
  NSQD_IP=$(docker inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-nsqd)
  docker run ${DEV_ENV_OPTS} \
    -e DRYCC_REDIS_ADDRS=${REDIS_IP}:6379 \
    -e DRYCC_NSQD_ADDRS=${NSQD_IP}:4150 \
    -it \
    ${DEV_ENV_IMAGE} \
    bash -c "test-cover.sh"
  stop-test-redis
  stop-test-nsqd
}

function test-unit() {
  start-test-redis test --cover --race -v
  start-test-nsqd
  REDIS_IP=$(docker inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-redis)
  NSQD_IP=$(docker inspect --format "{{ .NetworkSettings.IPAddress }}" test-logger-nsqd)
  docker run ${DEV_ENV_OPTS} \
    -e DRYCC_REDIS_ADDRS=${REDIS_IP}:6379 \
    -e  DRYCC_NSQD_ADDRS=${NSQD_IP}:4150 \
    -it \
    ${DEV_ENV_IMAGE} \
    bash -c "go test --cover --race -v -tags=testredis ./..."
  stop-test-redis
  stop-test-nsqd
}

"$@"
