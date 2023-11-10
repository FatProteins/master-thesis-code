#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/deploy.*/master-thesis-code/g')
DEPLOY_DIR="${PROJECT_ROOT}/deploy"
BFTSMART_PROJECT_ROOT="$HOME/java-projects/library-master"

while [ "$#" -gt 0 ]
do
  case "$1" in
  "--skip-install")
    SKIP_INSTALL=1
    ;;
  "--env-path")
    shift
    ENV_PATH="$1"
    ;;
  esac
  shift
done

if [ -z "${SKIP_INSTALL}" ]; then
  bash "${DEPLOY_DIR}/install-bftsmart.sh" "${BFTSMART_PROJECT_ROOT}"
else
  echo "Skipping BFT-Smart installation"
fi

if [ -z "${ENV_PATH}" ]; then
  echo "--env-path required - Path of .env file"
  exit 1
fi

. "${ENV_PATH}"

docker build -t "${BFTSMART_IMAGE_NAME}:${BFTSMART_IMAGE_VERSION}" -f "${DEPLOY_DIR}/Dockerfile-bftsmart" "${PROJECT_ROOT}"