# optimism-scanner
[Optimism] Chain Scanner Services

## Build a DockerHub

To build the docker image:

```bash

# Chain Scanner
docker build . --file Dockerfile.chain-scanner --tag enyalabs/optimism-chain-scanner:v1.0.0
docker push enyalabs/optimism-chain-scanner:v1.0.0

# Message Scanner
docker build . --file Dockerfile.L2ToL1Message-scanner --tag enyalabs/optimism-message-scanner:v1.0.0
docker push enyalabs/optimism-message-scanner:v1.0.0

```

## Chain Scanner

It scans the L2 and write the new block data, transaction data and receipt data into MySQL.

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
| `CHAIN_SCAN_INTERVAL` | No | 60,000 | Time (in milliseconds) to wait while scanning for new blocks. |

## L2 To L1 Message Scanner

It checks whether the message from L2 to L1 is finalized.

| Environment Variable      | Required? | Default Value                                   | Description                                                  |
| ------------------------- | --------- | ----------------------------------------------- | ------------------------------------------------------------ |
| `L1_NODE_WEB3_URL`        | No        | http://localhost:8545                           | HTTP endpoint for a Layer 1 (Ethereum) node.                 |
| `L2_NODE_WEB3_URL`        | No        | [http://localhost:9545](http://localhost:9545/) | HTTP endpoint for a Layer 2 (Optimism) Verifier node.        |
| `MYSQL_HOST_URL`          | No        | 127.0.0.1                                       | HTTP endpoint for MySQL.                                     |
| `MYSQL_PORT`              | No        | 3306                                            | Port for the MySQL connection.                               |
| `MYSQL_USERNAME`          | Yes       | N/A                                             | Name of the user to connect with.                            |
| `MYSQL_PASSWORD`          | Yes       | N/A                                             | The user's password.                                         |
| `MYSQL_DATABASE_NAME`     | No        | OMGXV1                                          | Name for the database.                                       |
| `ADDRESS_MANAGER_ADDRESS` | Yes       | N/A                                             | Contract address of the address manager                      |
| `L2_MESSENGER_ADDRESS`    | No        | 0x4200000000000000000000000000000000000007      | Contract address of L2 messenger                             |
| `DEPLOYER_PRIVATE_KEY`    | Yes       | N/A                                             | Private key for an account on Layer 1 (Ethereum) to be used to deploy contracts. |
| `MESSAGE_SCAN_INTERVAL`   | No        | 3,600,000                                       | Time (in milliseconds) to wait while scanning for new blocks. |
