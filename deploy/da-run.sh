rm -f "${DA_CONTAINER_SOCKET_PATH}"
/bin/envsubst < /thesis/config/fault-config.yml.tpl > /thesis/config/fault-config.yml
/thesis/da