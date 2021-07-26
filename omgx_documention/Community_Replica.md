- [Community Replica Service](#community-replica-service)
  * [Prerequisites](#prerequisites)
  * [Start Replica service](#start-replica-service)
    + [Configuration](#configuration)
    + [Start the docker](#start-the-docker)
    + [Common Errors](#common-errors)

# Community Replica Service

The `ops/docker-compose-replica-service.yml` docker-compose project runs a local replica of the OMGC Rinkeby l2geth. This service allows you run a local copy of the L2, which is useful for generating analyics for blockexplorers.

## Prerequisites

\- docker
\- docker-compose

## Start Replica service

### Configuration

Replace `INFURA_KEY` with your own key in [docker-compose-replica-service.yml](.ops/docker-compose-replica-service.yml). You can get a free Infura key from https://infura.io.

### Start the docker

Start the replica service via:

```bash
cd ops
docker-compose -f docker-compose-replica-service.yml up
```

This will pull two images from docker hub:

* [`data-tranport-layer`](https://hub.docker.com/layers/156092207/omgx/data-transport-layer/production-v1/images/sha256-07d4415aab46863b8c7996c1c40f6221f3ac3f697485ccc262a3a6f0478aa4fb?context=explore): service that indexes transaction data from the L1 chain and L2 chain

* [`replica`](https://hub.docker.com/layers/157390249/omgx/replica/production-v1/images/sha256-fc85c0db75352a911f49ba44372e087e54bd7123963f83a11084939f75581b37?context=explore): L2 geth node running in sync mode

### Common Errors

If you get this:

```bash
(node:1) UnhandledPromiseRejectionWarning: Error: could not detect network (event="noNetwork", code=NETWORK_ERROR, version=providers/5.1.0)
```

then you forgot to replace `INFURA_KEY` in this line: `DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT: https://rinkeby.infura.io/v3/INFURA_KEY` with your Infura key. Your Infura key will be a string like `c655138ed943455123456789123456789c`, so the final line will look something like this:

```bash
DATA_TRANSPORT_LAYER__L1_RPC_ENDPOINT: https://rinkeby.infura.io/v3/c655138ed943455123456789123456789c
```

