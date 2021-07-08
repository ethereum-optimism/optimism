# Replica Service

The docker-compose project runs a local replicate l2geth.

## Prerequisites

\- docker

\- docker-compose

## Start replica service

### Configuration

Replace `INFURA_KEY` with your own key in [docker-compose-replica-service.yml](./docker-compose-replica-service.yml).

### Start the docker

Start the replica service via:

```bash
cd ops/
docker-compose -f docker-compose-replica-service.yml up
```

This will pull the two images from docker hub:

* [`data-tranport-layer`](https://hub.docker.com/layers/156092207/omgx/data-transport-layer/production-v1/images/sha256-07d4415aab46863b8c7996c1c40f6221f3ac3f697485ccc262a3a6f0478aa4fb?context=explore): service that indexes transaction data from the L1 chain and L2 chain

* [`replica`](https://hub.docker.com/layers/157390249/omgx/replica/production-v1/images/sha256-fc85c0db75352a911f49ba44372e087e54bd7123963f83a11084939f75581b37?context=explore): L2 geth node running in sync mode