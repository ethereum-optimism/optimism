# Running a replica node

## Docker configuration

Here are instructions if you want to run boba erigon version as the replica node for OP Mainnet or Testnet.

### Get the data dir

The first step is to download the initial data for `op-erigon`. Thanks for the contribution from [Test in Prod](https://www.testinprod.io).

1. Download the correct data directory snapshot.

* [OP Mainnet](https://op-erigon-backup.mainnet.testinprod.io)
* [OP Goerli](https://op-erigon-backup.goerli.testinprod.io)

2. Create the data directory in `op-erigon` and fill it.

  ```bash
  mkdir op-erigon
  cd ./op-erigon
  mkdir erigon
  cd ./erigon
  tar xvf ~/[DIR]/op-erigon-goerli.tar
  ```

3. Create a shared secret (JWT token)

  ```bash
  cd op-erigon
  openssl rand -hex 32 > jwt.txt
  ```

>  This step is optional, but we highly recommand you to create your own JWT token.

### Create a .env file

Create a  `.env` file in `boba-community`. 

```
VERSION=
ETH1_HTTP=
```

> This step is optional, but we recommand you to use a latest release image for `VERSION`. Otherwise, it pulls the latest image.

### Modify volume location

The volumes of l2 should be modified to your file locations.

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
docker-compose -f docker-compose-op-goerli.yml up -d
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
