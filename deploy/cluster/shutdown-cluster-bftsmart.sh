#!/bin/bash

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/deploy.*/master-thesis-code/g')
DEPLOY_DIR="${PROJECT_ROOT}/deploy"
CLUSTER_DIR="${DEPLOY_DIR}/cluster"

. "${CLUSTER_DIR}/.env"

CLUSTER_SIZE=$(yq ".deployment.machines | length" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")

for (( i=0; i<CLUSTER_SIZE; i++ )); do
  CLUSTER_HOST_IP=$(yq -r ".deployment.machines[$i].host" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")
  CLUSTER_USER=$(yq -r ".deployment.machines[$i].user" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")
  echo "Shutting down bftsmart and da instances on ${CLUSTER_HOST_IP}"
  ssh "${CLUSTER_USER}@${CLUSTER_HOST_IP}" "docker rm -f ${PROJECT_NAME}-bftsmart-1 ${PROJECT_NAME}-da-1"
done