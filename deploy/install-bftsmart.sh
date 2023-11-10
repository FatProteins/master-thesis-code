#!/bin/bash

set -e

BFTSMART_PROJECT_PATH="$1"
PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')
BIN_DIR="${PROJECT_ROOT}/bin"

(
  cd "${BFTSMART_PROJECT_PATH}"
  rm -rf "${BFTSMART_PROJECT_PATH}/build"
  echo "Building BFT-Smart project"
  ./gradlew clean installDist
)

mkdir -p "${BIN_DIR}/bftsmart"
cp -r "${BFTSMART_PROJECT_PATH}/build" "${BIN_DIR}/bftsmart/"