# Overall Setup

Clone the repository, open it, and install nodejs packages with `yarn`:

```bash
git clone git@github.com:omgnetwork/optimism.git
cd optimism
yarn clean
yarn install
yarn build
```

Build and run the entire stack:

```bash
$ cd ops
$ BUILD=1 DAEMON=0 ./up_local.sh
```

Helpful commands:

* _Running out of space on your Docker, or having other having hard to debug issues_? Try running `docker system prune -a --volumes` and then rebuild the images. 
* _To (re)build individual base services_: `docker-compose build -- l2geth` 
* _To (re)build individual OMGX services_: `docker-compose -f "docker-compose.yml" build -- omgx_message-relayer-fast` Note: First you will have to comment out various dependencies in `docker-compose.yml`.

### Running unit tests

To run unit tests for a specific package:

```bash
cd packages/package-to-test
yarn test
```

### Running integration tests

```bash
cd integration-tests
yarn build:integration
yarn test:integration
```
