# Unipig
This project contains a simple website, aggregator service, validator service, and smart contracts to demonstrate an Optimistic Rollup PoC scaling the Uniswap smart conx   tract.


## Building
The project can be built by running
```sh
yarn run build
```

## Running the tests
Our tests make use of a combination of [`Mocha`](https://mochajs.org/) (a testing framework) and [`Chai`](https://www.chaijs.com/) (an assertion library) for testing.

Once built, run all tests with:

```sh
yarn test
```

## Deploying the smart contracts
You can deploy by running:

```sh
./exec/deployContracts.sh <environment>
```

The `environment` parameter tells the deployment script which config file to use (expected filename `.<environment>.env`).

For instance, to deploy the RollupChain contract using `.local.env` as the config file, you would run:

```sh
./exec/deployContracts.sh local
```

See `.env.contract.example` for more information.

## Running the Web Server
The basic website to demonstrate the unipig functionality can be run and will be available at `http://localhost:8080/`.
To run, make sure the project is built and run:
```sh
yarn run serve
```

## Running the Services 
### Configuration
The aggregator & validator expect an `.env` file that looks like the `/config/.env.service.example` in the same location. The idea is that there is some sensitive info there, so `.env` files are specifically ignored from git so that we never accidentally check in credentials.

### Running Aggregator
Make sure the project is built and run:
```sh
./exec/runAggregator.sh
```

### Running Validator
Make sure the project is built and run:
```sh
./exec/runValidator.sh
```

## Clearing Data
All data is stored within the `/build` directory at the moment, so if you'd like to blow away data, just run `yarn clean && yarn build` and run the aggregator / validator again. The DB for the aggregator & validator is leveldb, which persists to files in the `/build` directory that get blown away when you `yarn clean`