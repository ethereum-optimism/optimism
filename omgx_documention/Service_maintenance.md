# Service maintenance

## Docker containers

We have six main containers and five secondary containers that provide the monitor and subgraph services.

### Main containers

* [omgx/deployer-rinkeby](https://hub.docker.com/layers/omgx/deployer-rinkeby/production-v1/images/sha256-8ca509eb7a830ee862318225a2d5558f868d139a745edaff448ec3ccb90965e8?context=repo)

  It serves `addresses.json` and `state-dump.latest.json` files so that l2geth container and data-transport-layer can fetch the `ADDRESS_MANAGER_ADDRESS` and `ABI` of L1 pre-deployed contracts.

  Normally, this container uses about **15MB** memory.

* [omgx/l2geth](https://hub.docker.com/layers/156092279/omgx/l2geth/production-v1/images/sha256-d5f099b01629da9ca93af25705d326d90bb7d100695e0a66cc920871705ff890?context=repo)

  The l2geth container takes around 500MB memory. For safety and maintenance reasons, we should allocate **1GB** of memory to this service.

  > Note: We didn't see a large increase in memory usage of **l2geth** when we ran the performance test.

* [omgx/data-transport-layer](https://hub.docker.com/layers/156092207/omgx/data-transport-layer/production-v1/images/sha256-07d4415aab46863b8c7996c1c40f6221f3ac3f697485ccc262a3a6f0478aa4fb?context=repo)

  The data-transport-layer syncs L1 and provides the index service. It only uses about **50MB** of memory.

* [omgx/batch-submitter](https://hub.docker.com/layers/156091606/omgx/batch-submitter/production-v1/images/sha256-b3e61c1350b94cca73853867e1267e6f0e197ffbf7661f76c5c373e85eb3e70f?context=repo)

  The batch submitter submits TX and state root batches to CTC and SCC contracts. It takes about **100MB** of memory.

* [omgx/message-relayer](https://hub.docker.com/layers/156091959/omgx/message-relayer/production-v1/images/sha256-52ae4dbe41895c331ee3dc05955ad8c50c1319f91aaf3b4747d3ded2305382b4?context=repo) and [omgx/message-relayer-fast](https://hub.docker.com/layers/156091184/omgx/message-relayer-fast/production-v1/images/sha256-4e973130ca9cd5704ae3ce83f8c01682851b73835753268203bba91df7213167?context=repo)

  Both message relayers need at least **1GB** of memory. It will restart when they approach the memory usage around **3.5GB**. It's better to give them at least **4GB** memory, so they won't restart frequently.

### Secondary containers

Our main services won't be affected by these secondary services, so it's safe for them to reboot when they have any problems.

* Graph-node, postgres, ipfs
* omgx/monitoring
* omgx/dummy-transaction

## Port

We open the following the ports:

| **Port** |                **Purpose**                |                  **Routes**                   |                             URL                              | **Permission** |
| :------: | :---------------------------------------: | :-------------------------------------------: | :----------------------------------------------------------: | :------------: |
|   8545   |                  L2Geth                   |                       /                       |                 https://rinkeby.omgx.network                 |     Public     |
|   8081   |                 Deployer                  | /addresses.json<br />/state-dumps.latest.json |              https://rinkeby.omgx.network:8081               |     Public     |
|   8000   |            GraphQL HTTP server            |            /subgraphs/name/.../...            | https://graph.rinkeby.omgx.network <br />https://graph.rinkeby.omgx.network:8000 |     Public     |
|   8001   |                GraphQL WS                 |            /subgraphs/name/.../...            |           https://graph.rinkeby.omgx.network:8001            |     Public     |
|   8020   | JSON-RPC<br /> (for managing deployments) |                       /                       |           https://graph.rinkeby.omgx.network:8020            |    Private     |
|   8030   |       Subgraph indexing status API        |                   /graphql                    |           https://graph.rinkeby.omgx.network:8030            |     Public     |
|   8040   |            Prometheus metrics             |                   /metrics                    |           https://graph.rinkeby.omgx.network:8040            |     Public     |

## Memory usage and recommendation

|         Container         | Minimum memory usage | Recommanded memory allocation |
| :-----------------------: | :------------------: | :---------------------------: |
|   omgx/deployer-rinkeby   |         15MB         |             128MB             |
|        omgx/l2geth        |        500MB         |            **2GB**            |
| omgx/data-transport-layer |        100MB         |             512MB             |
|   omgx/batch-submitter    |         1GB          |            **2GB**            |
|   omgx/message-relayer    |         1GB          |            **4GB**            |
| omgx/message-relayer-fast |         1GB          |            **4GB**            |

> NOTE:
>
> `omgx/l2geth`: it's the most important service, so we should give as much memory as we can.
>
> `omgx/message-relayer-fast` and `omgx/message-relayer` : both message relayers can stop due the OOM issue. Giving it **4GB** memory can reduce the number of times that it needs to restart. 
>
> [**IMPORTANT**] When `omgx/message-relayer-fast` and `omgx/message-relayer` restart, they scans all L2 blocks and check if there is a cross domain message and if the message is relayed to L1. It takes about 5 mins to sync 4K L2 blocks.

## Possible errors

* [omgx/batch-submitter](https://hub.docker.com/layers/156091606/omgx/batch-submitter/production-v1/images/sha256-b3e61c1350b94cca73853867e1267e6f0e197ffbf7661f76c5c373e85eb3e70f?context=repo)

  The queued data in the `CTC-queue` contract might not match to the data of the L1 block. We have noticed the following situations:

  * `The timestamp of the queued element` >` timestamp of L1 block` and `block number of the queued element` === `block number of L1 block`.
  * `The timestamp of the queued element` >` timestamp of L1 block` and `block number of the queued element` > `block number of L1 block`.

  The second issue can be fixed by enabling the [AUTO_FIX_BATCH_OPTIONS_CONF](https://github.com/omgnetwork/optimism/blob/8fd511e608744f182f8a10e6fb5aa5d27f581860/packages/batch-submitter/src/exec/run-batch-submitter.ts#L241) to `fixMonotonicity`.

  Please comment out [fixedBatch.push(ele)](https://github.com/omgnetwork/optimism/blob/8fd511e608744f182f8a10e6fb5aa5d27f581860/packages/batch-submitter/src/batch-submitter/tx-batch-submitter.ts#L492) and enable [AUTO_FIX_BATCH_OPTIONS_CONF](https://github.com/omgnetwork/optimism/blob/8fd511e608744f182f8a10e6fb5aa5d27f581860/packages/batch-submitter/src/exec/run-batch-submitter.ts#L241) to `fixSkippedDeposits` for the first issue.

  > NOTE:
  >
  > You don't have to stop the batch-submitter in EC2 or ECS to fix the issue. Please add `.env` file to `packages/batch-submitter` and fix it via running the batch-submitter locally:
  >
  > ```bash
  > yarn build
  > yarn start
  > ```
  >
  > Once the local batch submitter pushes the correct queued elements to CTC, the production one will start to work.

* [omgx/message-relayer-fast](https://hub.docker.com/layers/156091184/omgx/message-relayer-fast/production-v1/images/sha256-4e973130ca9cd5704ae3ce83f8c01682851b73835753268203bba91df7213167?context=repo)

  It might has `MessageRelayerService._getStateBatchHeader` error when we ran the loading test. It can be fixed via restarting the service, but we don't know the root problem.

* [omgx/l2geth](https://hub.docker.com/layers/156092279/omgx/l2geth/production-v1/images/sha256-d5f099b01629da9ca93af25705d326d90bb7d100695e0a66cc920871705ff890?context=repo) [**IMPORTANT!!**]

  `omgx/l2geth` may have the incompatible genesis when we update the deployer.

  ```
   Fatal: Error starting protocol stack: database contains incompatible genesis
  ```

  We solve it by removing the old data. However, it's not a good way to solve it. It's related to #regensis topic.

## Regenesis

Two possible reasons that we need to do regenesis on L2geth.

1. When l2geth starts, it will fully sync with old data. If we have thousands of L2 blocks, it takes a few hours to do so.
2. When we deploy the new deployer contracts, probably we need to do regenesis.
