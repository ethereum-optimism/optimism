# Troubleshooting Runbook
This document provides a set of troubleshooting steps for common failure scenarios and how to resolve them.

## Indexer Troubleshooting

### Postgres failures
1. Unable to connect to postgres database:
* Verify that the database is healthy and accessible.
* Verify that the database credentials are correct.
* Verify that the database is not rate limiting connections.

2. Unable to insert or read rows:
* Verify that the database is healthy and accessible.
* Verify that the database credentials are correct.
* Verify that most recent`./migrations` schemas have been applied to the database.

### Header Traversal Failure
Header traversal is a client abstraction that allows the indexer to sequentially traverse the chain via batches of blocks. The following are some common failure modes and how to resolve them:
1. `the HeaderTraversal and provider have diverged in state`
* This error occurs when the indexer is operating on a different block state than the node. This is typically caused by network reorgs and is the result of `l1-confirmation-count` or `l2-confirmation-count` values being set too low. To resolve this issue, increase the confirmation count values and restart the indexer service.

2. `the HeaderTraversal's internal state is ahead of the provider`
* This error occurs when the indexer is operating on a block that the upstream provider does not have. This typically occurs when resyncing upstream node services. This issue typically resolves itself once the upstream node service is fully synced. If the problem persists, please file an issue.

### L1/L2 Processor Failures
The L1 and L2 processors are responsible for processing new blocks and system txs. Processor failures can spread and contaminate other downstream processors (i.e, bridge) as well. For example, if a L2 processor misses a block and fails to index a `MessagePassed` event, the bridge processor will fail to index the corresponding `WithdrawalProven` event and halt progress. The following are some common failure modes and how to resolve them:

1. A processor stops syncing arbitrarily due to a network/connectivity issue or syncing too slow.
* Verify that `batch-size` and `polling-interval` config values are sufficiently set in accordance with upstream node rate limits. If misconfigured, the indexer may be rate limited by the upstream node and fail to sync.
* Verify that the upstream dependency is healthy and accessible.

2. A processor failed to index a block or system tx. This should never happen as resiliency is built into the processor interaction logic, but if it does, the following investigations should be made:
* Verify that `preset` is set to proper L2 chain ID. If misconfigured, the indexer may be trying to index the wrong system contract addresses. There should be a log at startup that indicates the preset and system contract addresses being used.
* Verify that the upstream node dependency is healthy and accessible.
* Verify data tables to ensure that the block or system tx was indexed. If it wasn't indexed, please file an issue and resync the indexer (see below).

### Bridge Processor Failures
The bridge processor is responsible for indexing bridge tx and events (i.e, withdrawals, deposits). The bridge processor actively subscribes to new L1 block events where it waits for the prevalence of new batch submission epoch. Once detected, the processor scans the epoch blocks on L1 and L2 for initialized and finalized bridge events.

1. A finalized bridge event has no corresponding initiation (e.g, a detected finalized withdrawal has no corresponding proven event persisted in DB). The bridge process will halt indexing until the event is resolved.
* Verify that the indexer is incorrect by checking the L1/L2 block explorers for the corresponding withdrawal hash. The hash should be present on the `OptimismPortal` contract (`provenWithdrawals()`, `finalizedWithdrawals()`) as well as the L2 `L2ToL1MessagePasser` contract (`sentMessages()`). Foundry's `cast logs` [command](https://book.getfoundry.sh/reference/cast/cast-logs) provides a way to quickly do these lookups. If the hash is not present, the withdrawal was not initiated and the bridge processor is correct to halt indexing (TODO - Provide script that scans for correlated events on L2 using withdrawal hash). If the hash is present, the bridge processor is incorrect and should be investigated further. Please file an issue with the details of the investigation.

2. Bridge processor halting due to upstream L1/L2 processor failures. See above for troubleshooting steps for L1/L2 processor failures.
* Verify that the bridge processor is not halted due to upstream processor failures. If it is, remediate the upstream processor failures and restart the application. The bridge processor will be able to resume indexing from the last persisted epoch.

### Re-syncing
To resync a deployed indexer, the following steps should be taken:
1. Stop the running indexer service instance (especially if using blue/green deployment strategy). Deleting state while the application is running can cause data corruption since application startups begin operation at the most recent indexed state.
2. Delete the database state. This can be done by doing a cascading delete of both the l1/l2 `blocks` table. This will delete all blocks and associated data (i.e, system events, bridge txs). Since there's referential integrity between the blocks tables and all other table schema, the following commands will delete all data in the database:
```sql
TRUNCATE l1_block_headers CASCADE;
TRUNCATE l2_block_headers CASCADE;
```
3. Restart the indexer service. The indexer should detect that the database is empty and begin syncing from L2 genesis (i.e, `l2_start_height = 0`, `l1_start_height = l2_genesis_tx`). This can be verified by checking the logs for the following message:
```
no indexed state, starting from genesis
```

### Re-syncing bridge processor
To resync the bridge processor, the following steps should be taken:
1. Stop the running indexer service instance.
2. Delete the bridge processor's database state. This can be done by doing performing cascading delete of both the l1 `l1_transaction_deposits` and the `l2_transaction_withdrawals` tables:
```sql
TRUNCATE l1_transaction_deposits CASCADE;
TRUNCATE l2_transaction_withdrawals CASCADE;
```
3. Restart the indexer service.

## API Troubleshooting

### API is returning response errors
The API is a read-only service that does not modify the database. If the API is returning errors, it is likely due to the following reasons:
* Verify that the API is able to connect to the database. If the database is down, the API will return errors.
* Verify that http `timeout` env variable is set to a reasonable value. If the timeout is too low, the API may be timing out on requests and returning errors.
* Verify that rate limiting isn't enabled in request routing layer. If rate limiting is enabled, the API may be rate limited and returning errors.
* Verify that service isn't being overloaded by too many requests. If the service is overloaded, the API may be returning errors.
