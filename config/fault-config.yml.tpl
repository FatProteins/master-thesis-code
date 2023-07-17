unix-domain-socket-path: "${DA_CONTAINER_SOCKET_PATH}"
faults-enabled: true
actions:
  noop:
    probability: 0.5
  halt:
    probability: 0.1
    max-duration: 10
  pause:
    probability: 0.1
    max-duration: 10
    pause-command: "docker pause ${CONSENSUS_CONTAINER}"
    continue-command: "docker unpause ${CONSENSUS_CONTAINER}"
  stop:
    probability: 0.2
    max-duration: 10
    stop-command: "docker stop ${CONSENSUS_CONTAINER}"
    restart-command: "docker restart ${CONSENSUS_CONTAINER}"
  resend-last-message:
    probability: 0
    max-duration: 0