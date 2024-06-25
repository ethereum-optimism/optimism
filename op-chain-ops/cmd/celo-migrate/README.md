# Celo L2 Migration Script

## Overview

This script has two main sections. The first migrates Celo blocks to a format compatible with `op-geth`, and the second performs necessary state changes such as deploying L2 smart contracts.

### Block migration

The block migration itself has two parts: It first migrates the ancient / frozen blocks, which is all blocks before the last 90000. Because the ancients db is append-only, it copies these blocks into a new database after making the necessary transformations. The script then copies the rest of the chaindata directory (excluding `/ancients`) using the system level `rsync` command. All non-ancient blocks are then transformed in-place in the new db, leaving the old db unchanged.

### State migration

After all blocks have been migrated, the script performs a series of modifications to the state db. This is also done in-place in the `--new-db` directory. First, the state migration deploys the L2 smart contracts by iterating through the genesis allocs passed to the script and setting the nonce, balance, code and storage for each address accordingly, overwritting existing data if necessary. Finally, the state migration will commit the state changes to produce a new state root and create the first Cel2 block.

### Notes

Once the state changes are complete the migration is finished. The longest running section of the script is the ancients migration, and it can be resumed / skipped if interupted part way. The rest of the script cannot be resumed and will restart from the last migrated ancient block if interupted or re-run.

The script outputs a `rollup-config.json` file that is passed to the sequencer in order to start the L2 network.

See `--help` for how to run each portion of the script individually, along with other configuration options.

### Running the script

First, build the script by running

```bash
make celo-migrate
```

from the `op-chain-ops` directory.

You can then run the script as follows.

```bash
go run ./cmd/celo-migrate --help
```

NOTE: You will need `rsync` to run this script if it's not already installed

#### Running with local test setup (Alfajores / Holesky)

To test the script locally, we can migrate an alfajores database and use Holesky as our L1. The input files needed for this can be found in `./testdata`. The necessary smart contracts have already been deployed on Holesky.

##### Pull down the latest alfajores database snapshot

```bash
gcloud alpha storage cp gs://celo-chain-backup/alfajores/chaindata-latest.tar.zst alfajores.tar.zst
```

Unzip and rename

```bash
tar --use-compress-program=unzstd -xvf alfajores.tar.zst
mv chaindata ./data/alfajores_old
```

##### Generate test allocs file

The state migration takes in a allocs file that specifies the l2 state changes to be made during the migration. This file can be generated from the deploy config and l1 contract addresses by running the following from the `contracts-bedrock` directory.

```bash
CONTRACT_ADDRESSES_PATH=../../op-chain-ops/cmd/celo-migrate/testdata/deployment-l1-holesky.json \
DEPLOY_CONFIG_PATH=../../op-chain-ops/cmd/celo-migrate/testdata/deploy-config-holesky-alfajores.json \
STATE_DUMP_PATH=../../op-chain-ops/cmd/celo-migrate/testdata/l2-allocs-alfajores.json \
forge script ./scripts/L2Genesis.s.sol:L2Genesis \
--sig 'runWithStateDump()'
```

This should output the allocs file to `./testdata/l2-allocs-alfajores.json`. If you encounter difficulties with this and want to just continue testing the script, you can alternatively find the allocs file [here](https://gist.github.com/jcortejoso/7f90ba9b67c669791014661ccb6de81a).

##### Run script with test configuration

```bash
go run ./cmd/celo-migrate full \
--deploy-config ./cmd/celo-migrate/testdata/deploy-config-holesky-alfajores.json \
--l1-deployments ./cmd/celo-migrate/testdata/deployment-l1-holesky.json \
--l1-rpc https://ethereum-holesky-rpc.publicnode.com  \
--l2-allocs ./cmd/celo-migrate/testdata/l2-allocs-alfajores.json \
--outfile.rollup-config ./cmd/celo-migrate/testdata/rollup-config.json \
--old-db ./data/alfajores_old \
--new-db ./data/alfajores_new
```

The first time you run the script it should take ~5 minutes. The first part of the script will migrate ancient blocks, and will take the majority of the time.

During the ancients migration you can play around with stopping and re-running the script, which should always resume where it left off. If you run the script subsequent times after ancient migrations have been run, the script should skip ancient migrations and proceed to migrating non-ancient blocks quickly.

Note that partial migration progress beyond the ancient blocks (i.e. non-frozen blocks and state changes) will not be preserved between runs by default.

#### Running for Cel2 migration

##### Generate allocs file

You can generate the allocs file needed to run the migration with the following script in `contracts-bedrock`

```bash
CONTRACT_ADDRESSES_PATH=<PATH_TO_CONTRACT_ADDRESSES> \
DEPLOY_CONFIG_PATH=<PATH_TO_MY_DEPLOY_CONFIG> \
STATE_DUMP_PATH=<PATH_TO_WRITE_L2_ALLOCS> \
forge script scripts/L2Genesis.s.sol:L2Genesis \
--sig 'runWithStateDump()'
```

##### Dress rehearsal / pre-migration

To minimize downtime caused by the migration, node operators can prepare their Cel2 databases by running this script a day ahead of the actual migration. This will pre-populate the new database with most of the ancient blocks needed for the final migration, and will also serve as a dress rehearsal for the rest of the migration.

NOTE: The pre-migration should be run using a chaindata snapshot, rather than a db that is being used by a node. To avoid network downtime, we recommend that node operators do not stop any nodes in order to perform the pre-migration.

Node operators should inspect their migration logs after the dress rehearsal to ensure the migration completed succesfully and direct any questions to the Celo developer community on Discord before the actual migration.

##### Final migration

On the day of the actual cel2 migration, this script can be re-run using the same parameters as for the dress rehearsal but with the latest Celo Mainnet database snapshot as `--old-db`. The script will only need to migrate any ancient blocks frozen after the dress rehearsal, all non-frozen blocks, and state.

Unlike the pre-migration, the final migration can be run directly on the db used by the Celo node rather than a snapshot.
