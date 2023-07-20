#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/scripts.*/master-thesis-code/g')
BIN_DIR="${PROJECT_ROOT}/bin"

if [ -e "${BIN_DIR}/benchmark" ]; then
  echo "Found 'benchmark' in ${BIN_DIR}."
else
  echo "'benchmark' not found in ${BIN_DIR}. Installing..."
  . "${PROJECT_ROOT}/scripts/install-benchmark.sh"
fi

"${BIN_DIR}/benchmark" --target-leader --conns=10 --clients=1000 put --key-size=8 --sequential-keys --total=100000 --val-size=256