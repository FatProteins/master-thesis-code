#!/bin/bash

set -e

while [ "$#" -gt 0 ]
do
  case "$1" in
  "--disable-interrupt")
    DISABLE_INTERRUPT=1
    ;;
  esac
  shift
done

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/deploy.*/master-thesis-code/g')
DEPLOY_DIR="${PROJECT_ROOT}/deploy"
CLUSTER_DIR="${DEPLOY_DIR}/cluster"

. "${CLUSTER_DIR}/.env"

. "${CLUSTER_DIR}/shutdown-cluster-bftsmart.sh"

for (( i=0; i<CLUSTER_SIZE; i++ )); do
  CLUSTER_HOST_IP=$(yq -r ".deployment.machines[$i].host" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")
  CLUSTER_USER=$(yq -r ".deployment.machines[$i].user" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")
  echo "Starting bftsmart instance on ${CLUSTER_HOST_IP}"
  if [ -z "${DISABLE_INTERRUPT}" ]; then
    ssh "${CLUSTER_USER}@${CLUSTER_HOST_IP}" "REMOTE_DEPLOYMENT_DIR=${REMOTE_DEPLOYMENT_DIR} bash -s" -- < "${CLUSTER_DIR}/start-bftsmart-cluster.sh"
  else
    ssh "${CLUSTER_USER}@${CLUSTER_HOST_IP}" "REMOTE_DEPLOYMENT_DIR=${REMOTE_DEPLOYMENT_DIR} bash -s" -- < "${CLUSTER_DIR}/start-bftsmart-cluster.sh" "--disable-interrupt"
  fi
done