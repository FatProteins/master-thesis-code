#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/deploy.*/master-thesis-code/g')
DEPLOY_DIR="${PROJECT_ROOT}/deploy"
CLUSTER_DIR="${DEPLOY_DIR}/cluster"
TEMP_DEPLOYMENT_DIR="${CLUSTER_DIR}/temp"

. "${DEPLOY_DIR}/deploy-utils.sh"

mkdir -p "${TEMP_DEPLOYMENT_DIR}"

CLUSTER_SIZE=$(yq ".deployment.machines | length" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")
while [ "$#" -gt 0 ]
do
  case "$1" in
  "--skip-install")
    SKIP_INSTALL=1
    ;;
  "--skip-bftsmart-build")
    SKIP_BFTSMART_BUILD=1
    ;;
  "--skip-da-build")
    SKIP_DA_BUILD=1
    ;;
  "--disable-interrupt")
    DISABLE_INTERRUPT=1
    ;;
  esac
  shift
done

echo "Deploying on cluster with a cluster size of ${CLUSTER_SIZE}"

. "${CLUSTER_DIR}/.env"

if [ -z "${SKIP_DA_BUILD}" ]; then
  echo "Building da..."
  . "${DEPLOY_DIR}/build-da.sh" --env-path "${CLUSTER_DIR}/.env"
else
  echo "Skipping DA build"
fi

if [ -n "${SKIP_BFTSMART_BUILD}" ]; then
  echo "Skipping build of bftsmart image"
elif [ -z $SKIP_INSTALL ]; then
  . "${DEPLOY_DIR}/build-bftsmart.sh" --env-path "${CLUSTER_DIR}/.env"
else
  . "${DEPLOY_DIR}/build-bftsmart.sh" --skip-install --env-path "${CLUSTER_DIR}/.env"
fi

for (( i=0; i<CLUSTER_SIZE; i++ )); do
  CLUSTER_HOST_IP=$(yq -r ".deployment.machines[$i].host" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")
  BFTSMART_INITIAL_CLUSTER="${BFTSMART_INITIAL_CLUSTER}bftsmart_instance_$i=http://${CLUSTER_HOST_IP}:${BFTSMART_GRPC_PORT},"
done

BFTSMART_INITIAL_CLUSTER=${BFTSMART_INITIAL_CLUSTER%,}
echo "BFTSMART_INITIAL_CLUSTER=${BFTSMART_INITIAL_CLUSTER}"

declare -A INSTANCE_MAP
for (( i=0; i<CLUSTER_SIZE; i++ )); do
  INSTANCE_MAP[$i]="$(echo "$i" | tr '0-9' 'A-J')"
done

function upload_files() {
  INSTANCE_NUMBER="$1"
  INSTANCE_ID="${INSTANCE_NUMBER}"
  FROM_USER="$2"
  FROM_HOST="$3"
  cp "${CLUSTER_DIR}/.env" "${TEMP_DEPLOYMENT_DIR}/.env-cluster"
  CLUSTER_USER=$(yq -r ".deployment.machines[$INSTANCE_NUMBER].user" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")
  CLUSTER_HOST_IP=$(yq -r ".deployment.machines[$INSTANCE_NUMBER].host" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")

  {
    echo ""
    echo "BFTSMART_IMAGE_NAME=${BFTSMART_IMAGE_NAME}"
    echo "BFTSMART_IMAGE_VERSION=${BFTSMART_IMAGE_VERSION}"
    echo "HOST_IP=${CLUSTER_HOST_IP}"
    echo "REMOTE_DEPLOYMENT_DIR=${REMOTE_DEPLOYMENT_DIR}"
    echo "DA_VOLUME_PATH=${REMOTE_DEPLOYMENT_DIR}/volume/"
    echo "INSTANCE_NUMBER=${INSTANCE_NUMBER}"
    echo "CONSENSUS_CONTAINER_NAME=${PROJECT_NAME}-bftsmart-1"
    echo "DA_DISABLE_INTERRUPT=${DISABLE_INTERRUPT}"
    echo "CRASH_KEY=${INSTANCE_MAP[$INSTANCE_NUMBER]}"
    echo "INSTANCE_ID=${INSTANCE_ID}"
  } >> "${TEMP_DEPLOYMENT_DIR}/.env-cluster"

  echo "Copying files to ${CLUSTER_HOST_IP}"
  if [ -z "${SKIP_BFTSMART_BUILD}" ]; then
    if [ -z "${SKIP_DA_BUILD}" ]; then
      . "${CLUSTER_DIR}/copy-bftsmart-files-cluster.sh" --target-host "${CLUSTER_HOST_IP}" --target-user "${CLUSTER_USER}" --from-host "${FROM_HOST}" --from-user "${FROM_USER}"
    else
      . "${CLUSTER_DIR}/copy-bftsmart-files-cluster.sh" --skip-da-image-upload --target-host "${CLUSTER_HOST_IP}" --target-user "${CLUSTER_USER}" --from-host "${FROM_HOST}" --from-user "${FROM_USER}"
    fi
  else
    if [ -z "${SKIP_DA_BUILD}" ]; then
      . "${CLUSTER_DIR}/copy-bftsmart-files-cluster.sh" --skip-bftsmart-image-upload --target-host "${CLUSTER_HOST_IP}" --target-user "${CLUSTER_USER}" --from-host "${FROM_HOST}" --from-user "${FROM_USER}"
    else
      . "${CLUSTER_DIR}/copy-bftsmart-files-cluster.sh" --skip-da-image-upload --skip-bftsmart-image-upload --target-host "${CLUSTER_HOST_IP}" --target-user "${CLUSTER_USER}" --from-host "${FROM_HOST}" --from-user "${FROM_USER}"
    fi
  fi
  rm "${TEMP_DEPLOYMENT_DIR}/.env-cluster"
}

FIRST_CLUSTER_USER=$(yq -r ".deployment.machines[0].user" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")
FIRST_CLUSTER_HOST_IP=$(yq -r ".deployment.machines[0].host" "${CLUSTER_DIR}/deployment-cluster-bftsmart.yml")
upload_files "0"

for (( i=1; i<CLUSTER_SIZE; i++ )); do
  upload_files "$i" "${FIRST_CLUSTER_USER}" "${FIRST_CLUSTER_HOST_IP}"
done

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

rm -r "${TEMP_DEPLOYMENT_DIR}"
