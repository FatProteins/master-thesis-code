#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/scripts.*/master-thesis-code/g')
BIN_DIR="${PROJECT_ROOT}/bin"

echo "Downloading etcd module..."
GO111MODULE=off go get -u "go.etcd.io/etcd" || true

echo "Installing etcd benchmark tool..."
go build -o "${BIN_DIR}/benchmark" -C "${GOPATH}/src/go.etcd.io/etcd/tools/benchmark"
