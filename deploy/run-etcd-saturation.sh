#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')

for clients in 8704 9216 9728 10240 10752 11264 11776 12288 12800 13312 13824 14336 14848 15360 15872; do
  bash "${PROJECT_ROOT}/deploy/cluster/clean-cluster.sh"
  sleep 10
  ssh drotarmel@zs07.lab.dm.informatik.tu-darmstadt.de "cd new-saturation && ../saturation_measurement --endpoints=\"zs01.lab.dm.informatik.tu-darmstadt.de:2379,zs02.lab.dm.informatik.tu-darmstadt.de:2379,zs03.lab.dm.informatik.tu-darmstadt.de:2379,zs06.lab.dm.informatik.tu-darmstadt.de:2379,zs08.lab.dm.informatik.tu-darmstadt.de:2379\" --num-clients=${clients}"
done

bash "${PROJECT_ROOT}/deploy/cluster/shutdown-cluster.sh"