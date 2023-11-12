#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')

for clients in 1 4096; do
  for fault_leader in "true" "false" ; do
    for marker_value in "C" "S"; do
      bash "${PROJECT_ROOT}/deploy/cluster/clean-cluster.sh"
      sleep 10
      ssh drotarmel@zs07.lab.dm.informatik.tu-darmstadt.de "cd final-faults && ../fault_measurement --endpoints=\"zs01.lab.dm.informatik.tu-darmstadt.de:2379,zs02.lab.dm.informatik.tu-darmstadt.de:2379,zs03.lab.dm.informatik.tu-darmstadt.de:2379,zs06.lab.dm.informatik.tu-darmstadt.de:2379,zs08.lab.dm.informatik.tu-darmstadt.de:2379\" --num-clients=${clients} --fault-leader=${fault_leader} --marker-value=${marker_value}"
    done
  done
done

bash "${PROJECT_ROOT}/deploy/cluster/shutdown-cluster.sh"