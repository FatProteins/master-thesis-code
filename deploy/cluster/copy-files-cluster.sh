#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')
DEPLOY_DIR="${PROJECT_ROOT}/deploy"
CLUSTER_DIR="${DEPLOY_DIR}/cluster"
TEMP_DEPLOYMENT_DIR="${CLUSTER_DIR}/temp"

while [ "$#" -gt 0 ]
do
  case "$1" in
  "--target-host")
    shift
    TARGET_HOST="$1"
    ;;
  "--target-user")
    shift
    TARGET_USER="$1"
    ;;
  "--skip-image-upload")
    SKIP_IMAGE_UPLOAD=1
    ;;
  esac
  shift
done

. "${TEMP_DEPLOYMENT_DIR}/.env-cluster"

if [ -z "${TARGET_HOST}" ]; then
  echo "--target-host required - IP of remote to deploy on"
  exit 1
fi

if [ -z "${TARGET_USER}" ]; then
  echo "--target-user required - User to use for deployment on remote"
  exit 1
fi

ssh "${TARGET_USER}@${TARGET_HOST}" "mkdir -p etcd-deployment"
scp "${TEMP_DEPLOYMENT_DIR}/.env-cluster" "${CLUSTER_DIR}/docker-compose-cluster.yml" "${TARGET_USER}@${TARGET_HOST}:~/etcd-deployment/"

if [ -z "${SKIP_IMAGE_UPLOAD}" ]; then
  docker save -o "${TEMP_DEPLOYMENT_DIR}/etcd-image.tar" "${ETCD_IMAGE_NAME}:${ETCD_IMAGE_VERSION}"
  scp "${TEMP_DEPLOYMENT_DIR}/etcd-image.tar" "${TARGET_USER}@${TARGET_HOST}:~/etcd-deployment/"
else
  echo "Skipping upload of etcd image"
fi

ssh "${TARGET_USER}@${TARGET_HOST}" "docker load -i ~/etcd-deployment/etcd-image.tar"