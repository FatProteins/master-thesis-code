
rm -f "${TO_DA_CONTAINER_SOCKET_PATH}"
#rm "${FROM_DA_CONTAINER_SOCKET_PATH}"
/bin/envsubst < /thesis/config/fault-config.yml.tpl > /thesis/config/fault-config.yml
/thesis/da