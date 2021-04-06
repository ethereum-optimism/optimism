# Optimism Operations

## Building the images

It is recommended to use Docker BuildKit and Docker Compose CLI Build to
improve the container build times. You can do that as follows:

```
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_BUILDKIT=1
```

For your best experience, we also recommend aliasing 


## Bringing up a network

You can bring up an optimism network with the following command:

```
docker-compose up
```

You can use standard docker-compose commands for managing the containers, e.g. 
getting only the logs of `l2geth`: `docker-compose --follow l2geth`

## Docker Containers

Under `docker/` you will find all the docker containers we use. In particular,
`Dockerfile.monorepo` implements a multi-stage docker build in order to cache as many
installation steps as possible to save time.
