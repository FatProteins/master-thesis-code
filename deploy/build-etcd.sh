#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')
DEPLOY_DIR="${PROJECT_ROOT}/deploy"

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
  "${DEPLOY_DIR}/install-etcd.sh"
else
  echo "Skipping etcd installation"
fi

if [ -z "${ENV_PATH}" ]; then
  echo "--env-path required - Path of .env file"
  exit 1
fi

. "${ENV_PATH}"

docker build -t "${ETCD_IMAGE_NAME}:${ETCD_IMAGE_VERSION}" -f "${DEPLOY_DIR}/Dockerfile-etcd" "${PROJECT_ROOT}"
#docker build --build-arg CLUSTER_STRING="${CLUSTER_STRING}" -t "${ETCD_IMAGE_NAME}:${ETCD_IMAGE_VERSION}" .