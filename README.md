# master-thesis-code

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

For cluster deployment, use the script `deploy/cluster/deploy-etcd-cluster.sh`.
It uses the config file `deploy/cluster/deployment-cluster.yml` to determine the
remote hosts to deploy on and the ssh user to use.
Make sure the identity file is set up, or if password login is enabled, it needs to be
provided during deployment for each host.
