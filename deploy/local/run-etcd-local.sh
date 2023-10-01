#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis-code\/deploy.*/master-thesis-code/g')
DEPLOY_DIR="${PROJECT_ROOT}/deploy"
LOCAL_DIR="${DEPLOY_DIR}/local"

. "${DEPLOY_DIR}/deploy-utils.sh"

export CLUSTER_SIZE=2
while [ "$#" -gt 0 ]
do
  case "$1" in
  "--skip-install")
    SKIP_INSTALL=1
    ;;
  "--skip-etcd-build")
    SKIP_ETCD_BUILD=1
    ;;
  "--skip-da-build")
    SKIP_DA_BUILD=1
    ;;
  "--cluster-size")
    shift
    CLUSTER_SIZE=$1
    ;;
  "--disable-interrupt")
    DISABLE_INTERRUPT=1
    ;;
  esac
  shift
done

echo "Deploying locally with a cluster size of ${CLUSTER_SIZE}"
if [[ "${DISABLE_INTERRUPT}" == "1" ]]; then
  echo "Interrupt is turned OFF"
else
  echo "Interrupt is enabled"
fi

. "${LOCAL_DIR}/.env"

export ETCD_CLIENT_BASE_PORT=$ETCD_BASE_PORT
export ETCD_CLIENT_END_PORT=$((ETCD_CLIENT_BASE_PORT + CLUSTER_SIZE - 1))
export ETCD_GRPC_BASE_PORT=$((ETCD_CLIENT_END_PORT + 1))
export ETCD_GRPC_END_PORT=$((ETCD_GRPC_BASE_PORT + CLUSTER_SIZE - 1))

export HOST_IP=172.17.0.1

if [ -z "${SKIP_DA_BUILD}" ]; then
  echo "Building da..."
  . "${DEPLOY_DIR}/build-da.sh" --env-path "${LOCAL_DIR}/.env"
else
  echo "Skipping DA build"
fi

if [ -n "${SKIP_ETCD_BUILD}" ]; then
  echo "Skipping build of etcd image"
elif [ -z $SKIP_INSTALL ]; then
  echo "Building etcd..."
  . "${DEPLOY_DIR}/build-etcd.sh" --env-path "${LOCAL_DIR}/.env"
else
  echo "Building etcd..."
  . "${DEPLOY_DIR}/build-etcd.sh" --skip-install --env-path "${LOCAL_DIR}/.env"
fi

export ETCD_CLUSTER_TOKEN=12345678
export ETCD_CLUSTER_STATE=new
for (( j=0; j<CLUSTER_SIZE-1; j++ )); do
  DISCOVERY_PORT=$((ETCD_GRPC_BASE_PORT + j))
  ETCD_INITIAL_CLUSTER="${ETCD_INITIAL_CLUSTER}etcd_instance_$j=http://${HOST_IP}:${DISCOVERY_PORT},"
done

DISCOVERY_PORT=$((ETCD_GRPC_BASE_PORT + j))
ETCD_INITIAL_CLUSTER="${ETCD_INITIAL_CLUSTER}etcd_instance_$j=http://${HOST_IP}:${DISCOVERY_PORT}"
export ETCD_INITIAL_CLUSTER

for (( i=0; i<CLUSTER_SIZE; i++ )); do
  export PROJECT_NAME="consensus-$((i + 1))"
  export EXTERNAL_CLIENT_PORT=$((ETCD_CLIENT_BASE_PORT + i))
  export EXTERNAL_GRPC_PORT=$((ETCD_GRPC_BASE_PORT + i))
  export ETCD_INSTANCE_NAME="etcd_instance_$i"
  export CONSENSUS_CONTAINER_NAME="${PROJECT_NAME}-etcd-1"
  export DA_VOLUME_PATH="${PROJECT_ROOT}/volumes/volume-$i"
  export INSTANCE_NUMBER="$i"
  export DA_DISABLE_INTERRUPT="${DISABLE_INTERRUPT}"

#  if [[ "$i" == 0 ]]; then
#    cp "${LOCAL_DIR}/.env" "${PROJECT_ROOT}/.new-env"
#    printenv >> "${PROJECT_ROOT}/.new-env"
#  fi

#  if [[ "$i" -lt 4 ]]; then
    docker compose -p $PROJECT_NAME -f "${LOCAL_DIR}/docker-compose-local.yml" up -d da
    wait_for_da "${PROJECT_NAME}-da-1"
    docker compose -p $PROJECT_NAME -f "${LOCAL_DIR}/docker-compose-local.yml" up -d etcd
#  fi

done
