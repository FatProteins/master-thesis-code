#!/bin/bash

set -e

echo "Downloading etcd module"
GO111MODULE=off go get -u "go.etcd.io/etcd" || true

mkdir -p bin
BIN_PATH="$(pwd)/bin"

cd "${GOPATH}/src/go.etcd.io/etcd"
echo "Installing etcd"
./scripts/build.sh

cp -r ./bin/* "${BIN_PATH}"