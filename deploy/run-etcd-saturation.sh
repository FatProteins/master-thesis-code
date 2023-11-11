#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')

for clients in 1 2 4 8 16 32 64 128 256 512 1024 2048 4096 8192 16384 32768 65536 131072; do
  bash "${PROJECT_ROOT}/deploy/cluster/clean-cluster.sh"
  sleep 10
  ssh drotarmel@zs07.lab.dm.informatik.tu-darmstadt.de "cd new-saturation && ../saturation_measurement --endpoints=\"zs01.lab.dm.informatik.tu-darmstadt.de:2379,zs02.lab.dm.informatik.tu-darmstadt.de:2379,zs03.lab.dm.informatik.tu-darmstadt.de:2379,zs06.lab.dm.informatik.tu-darmstadt.de:2379,zs08.lab.dm.informatik.tu-darmstadt.de:2379\" --num-clients=${clients}"
done

bash "${PROJECT_ROOT}/deploy/cluster/shutdown-cluster.sh"