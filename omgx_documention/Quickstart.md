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
* _To (re)build individual OMGX services_: `docker-compose -f "docker-compose-omgx-services.yml" build -- omgx_message-relayer-fast` Note: First you will have to comment out various dependencies in `docker-compose-omgx-services.yml`.

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

## Front End Development

Start a local L1/L2. 

```bash
$ cd ops
$ BUILD=1 DAEMON=0 ./up_local.sh
```

Typically, you will only have to build everything once, and after that, you can save time by setting `BUILD` to `2`. Then, open a second terminal window and navigate to `packages/omgx/wallet-frontend`, and run

```bash
$ yarn get_artifacts #this will get all the contract artifacts - note that this will only work correctly if you ran `yarn build` at the top level per instructions
$ yarn build
$ yarn start
```

and the frontend should start up in a local browser. You can also develop on the Rinkeby testnet - in that case, you do not need to run a local L1/L2. If you would like to do that, just change the `.env` settings:

```bash
# This is for working on the wallet, pointed at the OMGX Rinkeby testnet
REACT_APP_INFURA_ID=
REACT_APP_ETHERSCAN_API=
REACT_APP_POLL_INTERVAL=20000
SKIP_PREFLIGHT_CHECK=true
REACT_APP_WALLET_VERSION=1.0.10
```
