#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')

for clients in 1 2 4 8 16 32 64 128 256 512 640 768 896 1024 1152 1280 1408 1536 1664 1792 1920 2048 4096; do
  bash "${PROJECT_ROOT}/deploy/cluster/clean-cluster-bftsmart.sh"
  sleep 10
  ssh drotarmel@zs08.lab.dm.informatik.tu-darmstadt.de "docker run -v /home/drotarmel/bftsmart-out:/thesis/out thesis-bftsmart-client:0.1 ${clients} X X"
done

bash "${PROJECT_ROOT}/deploy/cluster/shutdown-cluster-bftsmart.sh"