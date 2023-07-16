set -e

cd etcd-deployment
. ./.env-cluster

if docker compose; then
  COMPOSE_COMMAND="docker compose"
elif docker-compose; then
  COMPOSE_COMMAND="docker-compose"
else
  echo "ERROR: No docker compose command available on ${HOST_IP}."
  exit 1
fi

nohup bash -c "run_exp -m 'Testing Consensus Bandwidth CPU Thesis' -n 0 -t '0:10' -- '${COMPOSE_COMMAND} -p ${PROJECT_NAME} -f ./docker-compose-cluster.yml --env-file ./.env-cluster up etcd'" > etcd-log.log 2>&1 &
