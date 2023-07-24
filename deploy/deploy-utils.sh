function wait_for_da() {
  CONTAINER_NAME="$1"

  WAIT_COUNT=0
  while ! docker logs "${CONTAINER_NAME}" | grep -q "Ready"; do
    echo "${CONTAINER_NAME} not ready yet. Retrying in 5 seconds..."
    sleep 5
    WAIT_COUNT=$(( WAIT_COUNT + 1 ))
    if [[ ${WAIT_COUNT} -ge 10 ]]; then
      echo "Container ${CONTAINER_NAME} not ready after $(( WAIT_COUNT * 5 )) seconds. Aborting deployment..."
      exit 1
    fi
  done

  echo "${CONTAINER_NAME} is ready."
}