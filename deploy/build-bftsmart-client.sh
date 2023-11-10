#!/bin/bash

set -e

PROJECT_ROOT=$(pwd | sed 's/master-thesis\/deploy.*/master-thesis/g')

bash "${PROJECT_ROOT}/deploy/install-bftsmart.sh" /home/proteins/java-projects/library-master/
scp -r "${PROJECT_ROOT}/bin/bftsmart" drotarmel@zs08.lab.dm.informatik.tu-darmstadt.de:~/
scp "${PROJECT_ROOT}/deploy/Dockerfile-bftsmart-client" drotarmel@zs08.lab.dm.informatik.tu-darmstadt.de:~/
ssh drotarmel@zs08.lab.dm.informatik.tu-darmstadt.de "docker buildx build -t thesis-bftsmart-client:0.1 -f Dockerfile-bftsmart-client ."
#docker build -t thesis-bftsmart-client:0.1 -f "${PROJECT_ROOT}/deploy/Dockerfile-bftsmart-client" .
#docker save -o "${PROJECT_ROOT}/bftsmart-client-image.tar" thesis-bftsmart-client:0.1