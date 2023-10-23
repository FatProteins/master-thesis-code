unix-to-da-domain-socket-path: "${TO_DA_CONTAINER_SOCKET_PATH}"
unix-from-da-domain-socket-path: "${FROM_DA_CONTAINER_SOCKET_PATH}"
faults-enabled: true
education-mode: true
actions:
  noop:
  halt:
    max-duration: 10000
  pause:
    pause-command: "docker pause ${CONSENSUS_CONTAINER}"
    continue-command: "docker unpause ${CONSENSUS_CONTAINER}"
  stop:
    stop-command: "docker stop --signal SIGKILL ${CONSENSUS_CONTAINER}"
    restart-command: "docker restart ${CONSENSUS_CONTAINER}"
  resend-last-message:
    probability: 0.0
    max-duration: 0
  continue:
    continue-command: "docker unpause ${CONSENSUS_CONTAINER}"
  restart:
    restart-command: "docker restart ${CONSENSUS_CONTAINER}"