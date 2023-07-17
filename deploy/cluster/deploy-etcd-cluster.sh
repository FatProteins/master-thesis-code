#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/deploy.*/master-thesis-code/g')
DEPLOY_DIR="${PROJECT_ROOT}/deploy"
CLUSTER_DIR="${DEPLOY_DIR}/cluster"
TEMP_DEPLOYMENT_DIR="${CLUSTER_DIR}/temp"

mkdir -p "${TEMP_DEPLOYMENT_DIR}"

CLUSTER_SIZE=$(yq ".deployment.machines | length" "${CLUSTER_DIR}/deployment-cluster.yml")
while [ "$#" -gt 0 ]
do
  case "$1" in
  "--skip-install")
    SKIP_INSTALL=1
    ;;
  "--skip-build")
    SKIP_BUILD=1
    ;;
  esac
  shift
done
echo $CLUSTER_SIZE
. "${CLUSTER_DIR}/.env"


if [ -n "${SKIP_BUILD}" ]; then
  echo "Skipping build of etcd image"
elif [ -z $SKIP_INSTALL ]; then
  . "${DEPLOY_DIR}/build-etcd.sh" --env-path "${CLUSTER_DIR}/.env"
else
  . "${DEPLOY_DIR}/build-etcd.sh" --skip-install --env-path "${CLUSTER_DIR}/.env"
fi

for (( i=0; i<CLUSTER_SIZE; i++ )); do
  CLUSTER_HOST_IP=$(yq -r ".deployment.machines[$i].host" "${CLUSTER_DIR}/deployment-cluster.yml")
  ETCD_INITIAL_CLUSTER="${ETCD_INITIAL_CLUSTER}etcd_instance_$i=http://${CLUSTER_HOST_IP}:${ETCD_GRPC_PORT},"
done

ETCD_INITIAL_CLUSTER=${ETCD_INITIAL_CLUSTER%,}
echo "ETCD_INITIAL_CLUSTER=${ETCD_INITIAL_CLUSTER}"

for (( i=0; i<CLUSTER_SIZE; i++ )); do
  cp "${CLUSTER_DIR}/.env" "${TEMP_DEPLOYMENT_DIR}/.env-cluster"
  CLUSTER_HOST_IP=$(yq -r ".deployment.machines[$i].host" "${CLUSTER_DIR}/deployment-cluster.yml")
  CLUSTER_USER=$(yq -r ".deployment.machines[$i].user" "${CLUSTER_DIR}/deployment-cluster.yml")

  {
    echo ""
    echo "ETCD_IMAGE_NAME=${ETCD_IMAGE_NAME}"
    echo "ETCD_IMAGE_VERSION=${ETCD_IMAGE_VERSION}"
    echo "EXTERNAL_CLIENT_PORT=${ETCD_CLIENT_PORT}"
    echo "EXTERNAL_GRPC_PORT=${ETCD_GRPC_PORT}"
    echo "ETCD_INSTANCE_NAME=etcd_instance_$i"
    echo "ETCD_INITIAL_CLUSTER=${ETCD_INITIAL_CLUSTER}"
    echo "HOST_IP=${CLUSTER_HOST_IP}"
  } >> "${TEMP_DEPLOYMENT_DIR}/.env-cluster"
#  echo "ETCD_PROJECT_NAME=etcd-$((i + 1))" >> "${CLUSTER_DIR}/.env-cluster"
#  echo "EXTERNAL_CLIENT_PORT=$((ETCD_CLIENT_BASE_PORT + i))" >> "${CLUSTER_DIR}/.env-cluster"
#  echo "EXTERNAL_GRPC_PORT=$((ETCD_GRPC_BASE_PORT + i))" >> "${CLUSTER_DIR}/.env-cluster"
#  echo "ETCD_INSTANCE_NAME=etcd_instance_$i" >> "${CLUSTER_DIR}/.env-cluster"

  echo "Copying files to ${CLUSTER_HOST_IP}"
  if [ -z "${SKIP_BUILD}" ]; then
    . "${CLUSTER_DIR}/copy-files-cluster.sh" --target-host "${CLUSTER_HOST_IP}" --target-user "${CLUSTER_USER}"
  else
    . "${CLUSTER_DIR}/copy-files-cluster.sh" --target-host "${CLUSTER_HOST_IP}" --target-user "${CLUSTER_USER}" --skip-image-upload
  fi
  rm "${TEMP_DEPLOYMENT_DIR}/.env-cluster"
done

for (( i=0; i<CLUSTER_SIZE; i++ )); do
  CLUSTER_HOST_IP=$(yq -r ".deployment.machines[$i].host" "${CLUSTER_DIR}/deployment-cluster.yml")
  CLUSTER_USER=$(yq -r ".deployment.machines[$i].user" "${CLUSTER_DIR}/deployment-cluster.yml")
  echo "Starting etcd instance on ${CLUSTER_HOST_IP}"
  ssh "${CLUSTER_USER}@${CLUSTER_HOST_IP}" "bash -s" -- < "${CLUSTER_DIR}/start-cluster.sh"
done

rm -r "${TEMP_DEPLOYMENT_DIR}"
