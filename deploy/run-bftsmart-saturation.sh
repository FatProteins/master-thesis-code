#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')

#for clients in 1 2 4 8 16 32 64 128 256 512 640 768 896 1024 1152 1280 1408 1536 1664 1792 1920 2048 4096; do
#  bash "${PROJECT_ROOT}/deploy/cluster/clean-cluster-bftsmart.sh"
#  sleep 10
#  ssh drotarmel@zs08.lab.dm.informatik.tu-darmstadt.de "docker run -v /home/drotarmel/bftsmart-out:/thesis/out thesis-bftsmart-client:0.1 ${clients} X X"
#done

#bash "${PROJECT_ROOT}/deploy/build-bftsmart-client.sh" zf01 zf02 zf03 zf04

for clients in 4 8 16 32 64 128 256 512 640 768 896 1024 1152 1280 1408 1536 1664 1792 1920 2048 4096; do
  bash "${PROJECT_ROOT}/deploy/cluster/clean-cluster-bftsmart.sh"

  sleep 10

  per_instance=$((clients / 4))
  echo "Deploying ${per_instance} clients per instance."
  client_base=0
  for machine in "zf01" "zf02" "zf03"; do
    ssh drotarmel@${machine}.lab.dm.informatik.tu-darmstadt.de "docker run -d -v /home/drotarmel/bftsmart-saturation-four-machines:/thesis/out thesis-bftsmart-client:0.1 ${per_instance} X X ${client_base}"
    client_base=$((client_base + per_instance))
  done

  ssh drotarmel@zf04.lab.dm.informatik.tu-darmstadt.de "docker run -v /home/drotarmel/bftsmart-saturation-four-machines:/thesis/out thesis-bftsmart-client:0.1 ${per_instance} X X ${client_base}"

done

#bash "${PROJECT_ROOT}/deploy/cluster/shutdown-cluster-bftsmart.sh"