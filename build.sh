#!/usr/bin/env bash

set -e -u -o pipefail

VERSION=$(git rev-parse --short HEAD)
readonly VERSION
readonly IMAGE="xperimental/simple-ftp-resource:${VERSION}"

docker build -t "${IMAGE}" .
docker push "${IMAGE}"
