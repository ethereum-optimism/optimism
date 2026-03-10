# Running a replica node

## Docker configuration

Instructions for running a Boba replica node for Mainnet or Testnet.

### Choose your execution client

Boba supports multiple execution clients:

| Compose file | Execution | Consensus | Status |
|---|---|---|---|
| `docker-compose-boba-{network}-reth.yml` | **op-reth** | **op-node** | Recommended |
| `docker-compose-boba-{network}.yml` | op-erigon | op-node | Supported |
| `docker-compose-boba-{network}-geth.yml` | op-geth | op-node | Legacy |

> **Note:** The op-reth snapshots contain a pre-initialized reth database built from the op-geth state at the Bedrock migration block. Download, extract, and run — no manual `init-state` step is needed. To regenerate the database from scratch, see the [migration guide](scripts/geth-to-reth/README.md).

### Get the data dir

1. Download the initial data for your execution client.

- BOBA Sepolia

  The **erigon** db can be downloaded from [boba sepolia erigon db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-db.tgz).

  ```bash
  curl -o boba-sepolia-erigon-db.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-db.tgz
  ```

  The **geth** db can be downloaded from [boba sepolia geth db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db-20251002.tar.zst).

  ```bash
  curl -o boba-sepolia-geth-db.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db-20251002.tar.zst
  ```

  > Check the [BOBA Snapshots](https://docs.boba.network/for-developers/node-operators/snapshot-downloads) page for the correct checksum for the snapshot you've downloaded.

2. Extract the data Directory

   Once you've downloaded the database snapshot, you'll need to extract it to a directory on your machine. This will take some time to complete.

   ```bash
   tar xvf data.tgz
   ```

3. Create a shared secret (JWT token)

   ```bash
   openssl rand -hex 32 > jwt-secret.txt
   ```

### Create a .env file

Create a  `.env` file in `boba-community`.

```
ETH1_HTTP=
ETH2_HTTP=
ERIGON_VERSION=
OP_NODE_VERSION=
```

> `ETH2_HTTP` is mandatory as it is the L1 beacon endpoint. The other variables are optional, but we recommend using the latest release images. Otherwise, it will pull the image with the `latest` tag.

### Modify volume location

The volumes of l2 should be modified to your data directory location.

```yaml
l2:
  volumes:
    - ./jwt-secret.txt:/config/jwt-secret.txt
    - DATA_DIR:/db
```

### Start your replica node

For erigon + op-node (recommended):

```bash
docker compose -f docker-compose-boba-sepolia.yml up -d
```

### The initial synchronization

During the initial synchronization, you get log messages from the consensus node, and nothing else appears to happen.

```bash
INFO [08-04|16:36:07.150] Advancing bq origin                      origin=df76ff..48987e:8301316 originBehind=false
```

After a few minutes, the node finds the right batch and then it starts synchronizing.

```bash
INFO [08-04|16:36:01.204] Found next batch                         epoch=44e203..fef9a5:8301309 batch_epoch=8301309                batch_timestamp=1,673,567,518
INFO [08-04|16:36:01.205] generated attributes in payload queue    txs=2  timestamp=1,673,567,518
```

### Migrating from op-geth to op-reth

See the [migration guide](scripts/geth-to-reth/README.md) for instructions on generating a reth database from an existing op-geth or op-erigon node.

### Optional: Run the legacy node

Due to the anchorage migration, the new client does not support some RPC requests for the legacy blocks, such as `debug_transaction`. You can start the legacy node by running:

```bash
docker compose -f docker-compose-boba-sepolia-legacy.yml up -d
```

The legacy Geth database can be downloaded from the [snapshot page](https://docs.boba.network/for-developers/node-operators/snapshot-downloads).
