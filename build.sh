#!/usr/bin/env bash

set -eo pipefail
readonly IMAGE="pawel20987/simple-ftp-resource:latest"

docker build -t "${IMAGE}" .
docker push "${IMAGE}"
