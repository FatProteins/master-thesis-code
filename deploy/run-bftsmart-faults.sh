#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')

for clients in 1 1024; do
  for node in "A" "C" ; do
    for fault in "C" "S"; do
      bash "${PROJECT_ROOT}/deploy/cluster/clean-cluster-bftsmart.sh"
      sleep 10
      ssh drotarmel@zs08.lab.dm.informatik.tu-darmstadt.de "docker run -v /home/drotarmel/bftsmart-out:/thesis/out thesis-bftsmart-client:0.1 ${clients} ${node} ${fault}"
    done
  done
done

bash "${PROJECT_ROOT}/deploy/cluster/shutdown-bftsmart-cluster.sh"