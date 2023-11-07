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

. "${REMOTE_DEPLOYMENT_DIR}/deploy-utils.sh"
. "${REMOTE_DEPLOYMENT_DIR}/.env-cluster"
cd "${REMOTE_DEPLOYMENT_DIR}"

mkdir -p "${REMOTE_DEPLOYMENT_DIR}/volume"

if bash -c "docker compose > /dev/null 2>&1"; then
  COMPOSE_COMMAND="docker compose"
elif bash -c "docker-compose > /dev/null 2>&1"; then
  COMPOSE_COMMAND="docker-compose"
else
  echo "ERROR: No docker compose command available on ${HOST_IP}."
  exit 1
fi

if [ -z "${DISABLE_INTERRUPT}" ]; then
  bash -c "${COMPOSE_COMMAND} -p ${PROJECT_NAME} -f ${REMOTE_DEPLOYMENT_DIR}/docker-compose-cluster.yml --env-file ${REMOTE_DEPLOYMENT_DIR}/.env-cluster up -d da"
  wait_for_da "${PROJECT_NAME}-da-1"
else
  echo "Only starting etcd instances, because interrupts are disabled."
fi
bash -c "${COMPOSE_COMMAND} -p ${PROJECT_NAME} -f ${REMOTE_DEPLOYMENT_DIR}/docker-compose-cluster.yml --env-file ${REMOTE_DEPLOYMENT_DIR}/.env-cluster up -d etcd"

#echo "sleep infinity" > "${REMOTE_DEPLOYMENT_DIR}/start-thesis-experiment.sh"
#nohup bash -c "run_exp -m 'Testing Consensus Bandwidth CPU Thesis' -n 0 -t '0:10' -- 'bash ${REMOTE_DEPLOYMENT_DIR}/start-thesis-experiment.sh'" > "${REMOTE_DEPLOYMENT_DIR}/thesis-experiment.log" 2>&1 &
