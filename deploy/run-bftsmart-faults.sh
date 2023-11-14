#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')

#bash "${PROJECT_ROOT}/deploy/build-bftsmart-client.sh" zf01 zf02 zf03 zf04

for node in "A" "C" ; do
  for fault in "C" "S"; do
    bash "${PROJECT_ROOT}/deploy/cluster/clean-cluster-bftsmart.sh"
    sleep 10
    clients=256
    client_base=0
    for machine in "zf01" "zf02" "zf03"; do
      ssh drotarmel@${machine}.lab.dm.informatik.tu-darmstadt.de "docker run -d -v /home/drotarmel/bftsmart-out:/thesis/out thesis-bftsmart-client:0.1 ${clients} ${node} ${fault} ${client_base}"
      client_base=$((client_base + clients))
    done
    ssh drotarmel@zf04.lab.dm.informatik.tu-darmstadt.de "docker run -v /home/drotarmel/bftsmart-out:/thesis/out thesis-bftsmart-client:0.1 ${clients} ${node} ${fault} ${client_base}"
#      ssh drotarmel@zs08.lab.dm.informatik.tu-darmstadt.de "docker run -v /home/drotarmel/bftsmart-out:/thesis/out thesis-bftsmart-client:0.1 ${clients} ${node} ${fault}"
  done
done

bash "${PROJECT_ROOT}/deploy/cluster/shutdown-cluster-bftsmart.sh"