#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/deploy.*/master-thesis-code/g')
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
  "--skip-da-image-upload")
    SKIP_DA_IMAGE_UPLOAD=1
    ;;
  "--skip-etcd-image-upload")
    SKIP_ETCD_IMAGE_UPLOAD=1
    ;;
  "--from-host")
    shift
    FROM_HOST="$1"
    ;;
  "--from-user")
    shift
    FROM_USER="$1"
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

ssh "${TARGET_USER}@${TARGET_HOST}" "mkdir -p ${REMOTE_DEPLOYMENT_DIR}"
scp "${DEPLOY_DIR}/deploy-utils.sh" "${TEMP_DEPLOYMENT_DIR}/.env-cluster" "${CLUSTER_DIR}/docker-compose-cluster.yml" "${TARGET_USER}@${TARGET_HOST}:${REMOTE_DEPLOYMENT_DIR}"

if [ -z "${SKIP_IMAGE_UPLOAD}" ]; then
  if [ -z "${FROM_HOST}" ]; then
    echo "Copying from local"
    if [ -z "${SKIP_DA_IMAGE_UPLOAD}" ]; then
      docker save -o "${TEMP_DEPLOYMENT_DIR}/da-image.tar" "${DA_IMAGE_NAME}:${DA_IMAGE_VERSION}"
      scp "${TEMP_DEPLOYMENT_DIR}/da-image.tar" "${TARGET_USER}@${TARGET_HOST}:${REMOTE_DEPLOYMENT_DIR}"
    else
      echo "Skipping DA Image upload"
    fi
    if [ -z "${SKIP_ETCD_IMAGE_UPLOAD}" ]; then
      docker save -o "${TEMP_DEPLOYMENT_DIR}/etcd-image.tar" "${ETCD_IMAGE_NAME}:${ETCD_IMAGE_VERSION}"
      scp "${TEMP_DEPLOYMENT_DIR}/etcd-image.tar" "${TARGET_USER}@${TARGET_HOST}:${REMOTE_DEPLOYMENT_DIR}"
    else
      echo "Skipping etcd Image upload"
    fi
  else
    echo "Copying from ${FROM_HOST}"
    if [ -z "${SKIP_DA_IMAGE_UPLOAD}" ]; then
      ssh -t -o StrictHostKeyChecking=accept-new "${FROM_USER}@${FROM_HOST}" "scp ${REMOTE_DEPLOYMENT_DIR}/da-image.tar ${TARGET_USER}@${TARGET_HOST}:${REMOTE_DEPLOYMENT_DIR}"
    else
      echo "Skipping DA Image upload"
    fi
    if [ -z "${SKIP_ETCD_IMAGE_UPLOAD}" ]; then
      ssh -t -o StrictHostKeyChecking=accept-new "${FROM_USER}@${FROM_HOST}" "scp ${REMOTE_DEPLOYMENT_DIR}/etcd-image.tar ${TARGET_USER}@${TARGET_HOST}:${REMOTE_DEPLOYMENT_DIR}"
    else
      echo "Skipping etcd Image upload"
    fi
  fi

  if [ -z "${SKIP_DA_IMAGE_UPLOAD}" ]; then
    ssh "${TARGET_USER}@${TARGET_HOST}" "docker load -i ${REMOTE_DEPLOYMENT_DIR}/da-image.tar"
  fi
  if [ -z "${SKIP_ETCD_IMAGE_UPLOAD}" ]; then
    ssh "${TARGET_USER}@${TARGET_HOST}" "docker load -i ${REMOTE_DEPLOYMENT_DIR}/etcd-image.tar"
  fi

else
  echo "Skipping upload of etcd image"
fi
