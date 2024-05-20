# Running a replica node

## Docker configuration

Here are instructions if you want to run boba erigon version as the replica node for OP Mainnet or Testnet.

### Get the data dir

1. The first step is to download the initial data for `op-erigon`. 

- BOBA Sepolia

  The **erigon** db can be downloaded from the [boba sepolia erigon db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-db.tgz).

  ```bash
  curl -o boba-sepolia-erigon-db.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-erigon-db.tgz
  ```

  The **geth** db can be downloaded from [boba sepolia geth db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db.tgz).

  ```bash
  curl -o boba-sepolia-geth-db.tgz -sL https://boba-db.s3.us-east-2.amazonaws.com/sepolia/boba-sepolia-geth-db.tgz
  ```

- OP Mainnet

  The **erigon** db can be downloaded from [Test in Prod OP Mainnet](https://op-erigon-backup.mainnet.testinprod.io).

- OP Sepolia

  The **erigon** db can be downloaded from [optimism sepolia erigon db](https://boba-db.s3.us-east-2.amazonaws.com/sepolia/optimism-sepolia-erigon-db.tgz).

  Or you can generate the **erigon** db by downloading the genesis file from [Optimsim](https://networks.optimism.io/op-sepolia/genesis.json) and initialize the data directory with it.
  
  ```bash
  curl -o op-sepolia-genesis.json -sL https://networks.optimism.io/op-sepolia/genesis.json
  erigon init --datadir=/db genesis.json
  ```
  
  The erigon can be built from the [source](https://github.com/bobanetwork/v3-erigon) using `make erigon` .
  
  > You can verify the download by running the following command:
  >
  > ```
  > sha256sum boba-sepolia-erigon-db.tgz
  > ```
  >
  > You should see the following output
  >
  > ```
  > b887d2e0318e9299e844da7d39ca32040e3d0fb6a9d7abe2dd2f8624eca1cade  boba-sepolia-erigon-db.tgz
  > ```
  >
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
ERIGON_VERSION=
OP_NODE_VERSION=
ETH1_HTTP=
ETH2_HTTP=
```

> `ETH2_HTTP` is mandatory as it is the L1 beacon endpoint. The other variables are optional, but we recommend using the latest release images for `ERIGON_VERSION` and `OP_NODE_VERSION`. Otherwise, it will pull the image with the `latest` tag.

### Modify volume location

The volumes of l2 and op-node should be modified to your file locations.

```yaml
l2:
  volumes:
    - ./jwt-secret.txt:/config/jwt-secret.txt
    - DATA_DIR:/db
op-node:
  volumes:
  	- ./jwt-secret.txt:/config/jwt-secret.txt
```

### Start your replica node

```bash
docker-compose -f docker-compose-node.yml up -d
```

### The initial synchornization

During the initial synchonization, you get log messages from `op-node`, and nothing else appears to happen.

```bash
INFO [08-04|16:36:07.150] Advancing bq origin                      origin=df76ff..48987e:8301316 originBehind=false
```

After a few minutes, `op-node` finds the right batch and then it starts synchronizing. During this synchonization process, you get log messags from `op-node`.

```bash
INFO [08-04|16:36:01.204] Found next batch                         epoch=44e203..fef9a5:8301309 batch_epoch=8301309                batch_timestamp=1,673,567,518
INFO [08-04|16:36:01.205] generated attributes in payload queue    txs=2  timestamp=1,673,567,518
INFO [08-04|16:36:01.265] inserted block                           hash=ee61ee..256300 number=4,069,725 state_root=a582ae..33a7c5 timestamp=1,673,567,518 parent=5b102e..13196c prev_randao=4758ca..11ff3a fee_recipient=0x4200000000000000000000000000000000000011 txs=2  update_safe=true
```

### Optional: Run the legacy node

Due to the anchorage migration, the new client does not support some RPC requests for the legacy blocks, such as `debug_transaction`. You can start the legacy node by running:

```bash
docker-compose -f docker-compose-boba-sepolia-legacy.yml
```

The legacy Geth database can be downloaded from the [snapshot page](https://docs.boba.network/for-developers/node-operators/snapshot-downloads).
