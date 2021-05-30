# OMGX Integration

A single repository that spins up a local OMGX system based on dockers. This repo is a good starting point for becoming familiar with the system, run integration tests, and deploy your contracts to a local net. 

## Requirements

- [docker](https://docs.docker.com/get-docker/)
  - At least version 19.03.12
- [docker-compose](https://docs.docker.com/compose/install/)
  - At least version 1.27.3
- Recommended Docker memory allocation of >=8 GB.

## Setup

```bash

$ git clone git@github.com:omgnetwork/omgx_integration.git
$ cd omgx_integration

```

Docker will automatically use local images (if found), or Docker will pull the latest ones from [Dockerhub](https://hub.docker.com/u/omgx). To pull the latest images, run:

```bash

$ docker-compose -f docker-compose-local.yml pull

```

To run the entire system (L1 + L2) locally:

```bash

$ ./up_local.sh

```

This will spin up both chains, deploy all the needed contracts, and fund several test wallets. 

## Integration Testing

Please see the `omgx_integration-test` repo (coming soon). These should all exit with code 0.