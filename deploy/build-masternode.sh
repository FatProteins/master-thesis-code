#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/deploy.*/master-thesis-code/g')

go build -o "${PROJECT_ROOT}/bin/masternode" "${PROJECT_ROOT}/masternode/main.go"
