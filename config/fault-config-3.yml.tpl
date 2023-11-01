unix-to-da-domain-socket-path: "${TO_DA_CONTAINER_SOCKET_PATH}"
unix-from-da-domain-socket-path: "${FROM_DA_CONTAINER_SOCKET_PATH}"
faults-enabled: true
actions:
  noop:
    probability: 1.0
  halt:
    probability: 0.0
    halt-command: "docker exec ${CONSENSUS_CONTAINER} tc qdisc add dev eth0 root netem delay 90ms"
  unhalt:
    probability: 0.0
    unhalt-command: "docker exec ${CONSENSUS_CONTAINER} tc qdisc delete dev eth0 root netem delay 90ms"
  pause:
    probability: 0.0
    max-duration: 1000
    pause-command: "docker pause ${CONSENSUS_CONTAINER}"
    continue-command: "docker unpause ${CONSENSUS_CONTAINER}"
  stop:
    probability: 0.0
    max-duration: 0
    stop-command: "docker stop --signal SIGKILL ${CONSENSUS_CONTAINER}"
    restart-command: "docker restart ${CONSENSUS_CONTAINER}"
  resend-last-message:
    probability: 0.0
    max-duration: 0