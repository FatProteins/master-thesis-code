#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')

for clients in 1 4096; do
  bash "${PROJECT_ROOT}/deploy/cluster/clean-cluster.sh"
  sleep 10
  ssh drotarmel@zs07.lab.dm.informatik.tu-darmstadt.de "cd final-fit-measurement-external && ../saturation_measurement --endpoints=\"zs01.lab.dm.informatik.tu-darmstadt.de:2379,zs02.lab.dm.informatik.tu-darmstadt.de:2379,zs03.lab.dm.informatik.tu-darmstadt.de:2379,zs06.lab.dm.informatik.tu-darmstadt.de:2379,zs08.lab.dm.informatik.tu-darmstadt.de:2379\" --num-clients=${clients}"
done

bash "${PROJECT_ROOT}/deploy/cluster/shutdown-cluster.sh"