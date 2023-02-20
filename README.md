
# Drycc Logger
[![Build Status](https://woodpecker.drycc.cc/api/badges/drycc/logger/status.svg)](https://woodpecker.drycc.cc/drycc/logger)
[![codecov.io](https://codecov.io/github/drycc/logger/coverage.svg?branch=main)](https://codecov.io/github/drycc/logger?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/drycc/logger)](https://goreportcard.com/report/github.com/drycc/logger)

Drycc - A Fork of Drycc Workflow

Drycc (pronounced DAY-iss) Workflow is an open source Platform as a Service (PaaS) that adds a developer-friendly layer to any [Kubernetes](http://kubernetes.io) cluster, making it easy to deploy and manage applications on your own servers.

![Drycc Graphic](https://getdrycc.blob.core.windows.net/get-drycc/drycc-graphic-small.png)

For more information about the Drycc Workflow, please visit the main project page at https://github.com/drycc/workflow.

We welcome your input! If you have feedback, please [submit an issue][issues]. If you'd like to participate in development, please read the "Development" section below and [submit a pull request][prs].

## Description
A system logger for use in the [Drycc Workflow](https://drycc.com/workflow/) open source PaaS.

The new v2 logger implementation has seen a simplification from the last rewrite. While it still uses much of that code it no longer depends on `etcd`. Instead, we will use kubernetes service discovery to determine where logger is running.

We have also decided to not use `logspout` as the mechanism to get logs from each container to the `logger` component. Now we will use [fluentd](http://fluentd.org) which is a widely supported logging framework with hundreds of plugins. This will allow the end user to configure multiple destinations such as Elastic Search and other Syslog compatible endpoints like [papertrail](http://papertrailapp.com).

## Configuration
The following environment variables can be used to configure logger:

| Name                                   | Default Value |
|----------------------------------------|---------------|
| STORAGE_ADAPTER                        | "redis"       |
| NUMBER_OF_LINES (per app)              | "1000"        |
| AGGREGATOR_TYPE                        | "redis"       |
| DRYCC_REDIS_STREAM                     | logs          |
| DRYCC_REDIS_STREAM_GROUP               | logger        |
| AGGREGATOR_STOP_TIMEOUT_SEC            | 1             |
| DRYCC_REDIS_ADDRS                      | ":6379"       |
| DRYCC_REDIS_PASSWORD                   | ""            |
| DRYCC_REDIS_PIPELINE_LENGTH            | 50            |
| DRYCC_REDIS_PIPELINE_TIMEOUT_SECONDS   | 1             |

## Development
The only assumption this project makes about your environment is that you have a working docker host to build the image against.

### Building binary and image
To build the binary and image run the following make command:

```console
IMAGE_PREFIX=myaccount make build
DEV_REGISTRY=myhost:5000 make build
```

### Pushing the image
The makefile assumes that you are pushing the image to a remote repository like quay or dockerhub. So you will need to supply the `REGISTRY` environment variable.

```console
IMAGE_PREFIX=myaccount make push
DEV_REGISTRY=myhost:5000 make push
```

### Kubernetes interactions
* `make install` - Install the recently built docker image into the kubernetes cluster
* `make upgrade` - Upgrade a currently installed image
* `make uninstall` - Uninstall logger from a kubernetes cluster

### Architecture Diagram

```
┌──────────┐             ┌─────────┐  logs/metrics   ┌───────────────┐
│ App Logs │──Log File──▶│ Fluentd │─────Topics─────▶│ Redis XStream │
└──────────┘             └─────────┘                 └───────────────┘
                                                             │
                                                             │
                         ┌─────────┐       logs/xstream      │
                         │  Logger │◀----------Read----------┘
                         └─────────┘
```

[issues]: https://github.com/drycc/logger/issues
[prs]: https://github.com/drycc/logger/pulls
[workflow]: https://github.com/drycc/workflow
