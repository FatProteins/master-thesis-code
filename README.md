# master-thesis-code

## Dependencies
`go v1.20.5`

`Docker`

`Java 21` (for BFT-SMaRt)

## Deployment

All files for deployment are located in the subdirectory `deploy`.
Deployment scripts are provided for local and cluster deployment (TU Darmstadt DM Cluster).
All scripts should be run with the project root as working directory.

### Local

For local deployment, use the script `deploy/local/run-etcd-local.sh`.
It will also install dependencies and build the required images before deploying them.
After the first usage, if no rebuild is required, flags can be passed to skip rebuilding.
The following flags are accepted:

```
--skip-install      - Skips installing etcd, but still rebuilds the docker image.

--skip-etcd-build   - Skips installing and building etcd.

--skip-da-build     - Skips building the da application and image.

--cluster-size <n>  - Deploys <n> etcd nodes and da instances. (default 2)
```

### Cluster

For cluster deployment, use the script `deploy/cluster/deploy-etcd-cluster.sh` for etcd
and `deploy/cluster/deploy-cluster-bftsmart.sh` for BFT-SMaRt.
It uses the config file `deploy/cluster/deployment-cluster.yml` to determine the
remote hosts to deploy on and the ssh user to use.
Make sure the identity file is set up, or if password login is enabled, it needs to be
provided during deployment for each host.

## Experiments
Checkout branch `performance-experiments` to
- run scripts for performance experiments: produces CSV files with client request
timings
- deploy etcd cluster or BFT-SMaRt cluster for fault experiments

Last relevant commit before thesis submission for the experiments: a4eca30b8966fc87bcc15ea8b688c5b8cbd1e724

To reproduce the fault experiments from the thesis, make sure install the forked projects
instead of the original implementation:
- https://github.com/FatProteins/etcd-fork
- https://github.com/FatProteins/bft-smart-fork

Make sure to change `REMOTE_DEPLOYMENT_DIR` in `deploy/cluster/.env` according to your remote directory.
For the BFT-SMaRt client, a Dockerfile is provided `Dockerfile-bftsmart-client`. For etcd, the client is run
without docker.

### Consensus UI
Checkout branch `consensus-ui` and run the script `deploy/local/run-etcd-education.sh`.
This deploys the backend locally with an etcd cluster.
For the frontend, please refer to https://github.com/FatProteins/consensus-ui.

Last relevant commit before thesis submission for the Consensus UI: e817b1e31f3d672012d0d9d2033bd80d79096a8d