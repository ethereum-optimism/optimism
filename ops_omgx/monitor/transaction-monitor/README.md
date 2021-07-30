# Transaction Monitor

## Run locally
```bash
# Install dependecies and copy artifacts
yarn install
yarn copy
# Start transaction monitor service
yarn start

```

## Build a DockerHub

To build the docker image(note you only have to run `yarn copy` once):

```bash
# Install dependecies and copy artifacts
yarn install
yarn copy

# Build Transaction Monitor Image
docker build . --file ./Dockerfile --tag omg/transaction-monitor:production-v1
docker push omg/transaction-monitor:production-v1
```
You will have to create the `.env-docker` file and populate it with your
credentials. Look at `.env-docker.example` for an example.
You will use this file to set environment variables:
```bash
# Set environment variables
source ./.env-docker
```
To run the docker image:
```bash
# Start Transaction Monitor
docker-compose up
```


## Transaction Monitor

It scans the L2 and write the new block data, transaction data and receipt data into MySQL and checks whether the message from L2 to L1 has been finalized.

| Environment Variable        | Required? | Default Value         | Description            |
| -----------                 | --------- | -------------         | -----------           |
| `L1_NODE_WEB3_URL`        | No        | http://localhost:8545                           | HTTP endpoint for a Layer 1 (Ethereum) node.                 |
| `L2_NODE_WEB3_URL`        | No        | [http://localhost:9545](http://localhost:9545/) | HTTP endpoint for a Layer 2 (Optimism) Verifier node.        |
| `MYSQL_HOST_URL` | No        | 127.0.0.1    | HTTP endpoint for MySQL. |
| `MYSQL_PORT`   | No        | 3306         | Port for the MySQL connection. |
| `MYSQL_USERNAME` | Yes      | N/A              | Name of the user to connect with. |
| `MYSQL_PASSWORD` | Yes     | N/A                  | The user's password. |
| `MYSQL_DATABASE_NAME` | No        | OMGXV1               | Name for the database. |
| `ADDRESS_MANAGER_ADDRESS` | Yes      | N/A                 | Contract address of the address manager |
| `L2_MESSENGER_ADDRESS` | No        | 0x4200000000000000000000000000000000000007 | Contract address of L2 messenger |
| `DEPLOYER_PRIVATE_KEY` | Yes | N/A | Private key for an account on Layer 1 (Ethereum) to be used to deploy contracts. |
| `TRANSACTION_MONITOR_INTERVAL` | No | 60,000 | Time (in milliseconds) to wait while scanning for new blocks. |
| `CROSS_DOMAIN_MESSAGE_MONITOR_INTERVAL`   | No        | 300,000                                       | Time (in milliseconds) to wait while updating message receipts. |
