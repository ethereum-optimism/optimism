# Rollup Services

Contains executable scripts for the various rollup microservices contained in [rollup-core](../rollup-core)

## Configuration
All microservice configuration, including _which_ microservices get started, is done via environment variables.

### Which Microservices to Run
Below is a list of environment variables to set _to something_ in order to run the associated microservices:
* L1 Chain Data Persister: `RUN_L1_CHAIN_DATA_PERSISTER=1`
* L2 Chain Data Persister: `RUN_L2_CHAIN_DATA_PERSISTER=1`
* Geth Submission Queuer: `RUN_GETH_SUBMISSION_QUEUER=1`
* Queued Geth Submitter: `RUN_QUEUED_GETH_SUBMITTER=1`
* Canonical Transaction Chain Batch Creator: `RUN_CANONICAL_CHAIN_BATCH_CREATOR=1`
* Canonical Transaction Chain Batch Submitter: `RUN_CANONICAL_CHAIN_BATCH_SUBMITTER=1`
* State Commitment Chain Batch Creator: `RUN_STATE_COMMITMENT_CHAIN_BATCH_CREATOR=1`
* State Commitment Chain Batch Submitter: `RUN_STATE_COMMITMENT_CHAIN_BATCH_SUBMITTER=1`
* Fraud Detector: `RUN_FRAUD_DETECTOR=1`

### Service Configuration

#### Common Dependencies:
Postgres (needed for all):
* `POSTGRES_HOST` - (Required) The host DNS entry / IP for the postgres DB
* `POSTGRES_PORT` - The port for postgres (defaults to 5432)
* `POSTGRES_USER` - (Required) The user to use to connect to the db
* `POSTGRES_PASSWORD` - (Required) The password to use to connect to the db
* `POSTGRES_DATABASE` - The database name to connect to (defaults to `rollup`)
* `POSTGRES_CONNECTION_POOL_SIZE` - The connection pool size for postgres (defaults to 20)
* `POSTGRES_USE_SSL` - Set to anything to indicate that SSL should be used in the connection

L1 Node (needed for some):
* If Infura:
  * `L1_NODE_INFURA_NETWORK` - (Required) The Infura network for the connection to the node
  * `L1_NODE_INFURA_PROJECT_ID` - (Required) The Infura project ID for the connection to the node
* If not Infura:
  * `L1_NODE_WEB3_URL` - (Required) The URL of the L1 node
  
L2 Node (needed for some):
* `L2_NODE_WEB3_URL` - (Required) The URL of the L2 node

#### L1 Chain Data Persister
* Postgres - See Postgres section above
* L1 Node - See L1 Node section above
* Contracts:
  * `L1_TO_L2_TRANSACTION_QUEUE_CONTRACT_ADDRESS` - (Required) The address of the L1ToL2TransactionQueue contract
  * `SAFETY_TRANSACTION_QUEUE_CONTRACT_ADDRESS` - (Required) The address of the SafetyTransactionQueue contract
  * `CANONICAL_TRANSACTION_CHAIN_CONTRACT_ADDRESS` - (Required) The address of the CanonicalTransactionChain contract
  * `STATE_COMMITMENT_CHAIN_CONTRACT_ADDRESS` - (Required) The address of the StateCommitmentChain contract
* `L1_CHAIN_DATA_PERSISTER_DB_PATH` - (Required) The filepath where to locate (or create) the L1 Chain Data Persister LevelDB database
* `L1_EARLIEST_BLOCK` - (Required) The earliest block to sync on L1 to start persisting data
* `FINALITY_DELAY_IN_BLOCKS` - (Required) The number of blocks required to consider a submission final on L1

#### L2 Chain Data Persister
* Postgres - See Postgres section above
* L2 Node - See L2 Node section above
* `L2_CHAIN_DATA_PERSISTER_DB_PATH` - (Required) The filepath where to locate (or create) the L2 Chain Data Persister LevelDB database

#### Geth Submission Queuer
* Postgres - See Postgres section above
* `IS_SEQUENCER_STACK` - (Required) Set if this is queueing Geth submissions for a sequencer (and not _just_ a verifier)
* `GETH_SUBMISSION_QUEUER_PERIOD_MILLIS` - The period in millis at which the GethSubmissionQueuer should attempt to queue an L2 Geth submission (defaults to 10,000)

#### Queued Geth Submitter
* Postgres - See Postgres section above
* L2 Node - See L2 Node section above
* `QUEUED_GETH_SUBMITTER_PERIOD_MILLIS` - The period in millis at which the QueuedGethSubmitter should attempt to send L2 Geth submissions (defaults to 10,000)

#### Canonical Transaction Chain Batch Creator
* Postgres - See Postgres section above
* `CANONICAL_CHAIN_MIN_BATCH_SIZE` - The minimum batch size to build -- if fewer than this number of transactions are ready, a batch will not be created (defaults to 10)
* `CANONICAL_CHAIN_MAX_BATCH_SIZE` - The maximum batch size to build -- if more than this number of transactions are ready, they will be split into multiple batches of at most this size (defaults to 100)
* `CANONICAL_CHAIN_BATCH_CREATOR_PERIOD_MILLIS` - The period in millis at which the CanonicalChainBatchCreator should attempt to create Canonical Chain Batches (defaults to 10,000)

#### Canonical Transaction Chain Batch Submitter
* Postgres - See Postgres section above
* L1 Node - See L1 Node section above
* `CANONICAL_TRANSACTION_CHAIN_CONTRACT_ADDRESS` - (Required) The address of the CanonicalTransactionChain contract
* `L1_SEQUENCER_PRIVATE_KEY` - (Required) The private key to use to submit Sequencer Transaction Batches
* `FINALITY_DELAY_IN_BLOCKS` - (Required) The number of blocks required to consider a submission final on L1
* `CANONICAL_CHAIN_BATCH_SUBMITTER_PERIOD_MILLIS` - The period in millis at which the CanonicalChainBatchCreator should attempt to create Canonical Chain Batches (defaults to 10,000)

#### State Commitment Chain Batch Creator
* Postgres - See Postgres section above
* `STATE_COMMITMENT_CHAIN_MIN_BATCH_SIZE` - The minimum batch size to build -- if fewer than this number of transactions are ready, a batch will not be created (defaults to 10)
* `STATE_COMMITMENT_CHAIN_MAX_BATCH_SIZE` - The maximum batch size to build -- if more than this number of transactions are ready, they will be split into multiple batches of at most this size (defaults to 100)
* `STATE_COMMITMENT_CHAIN_BATCH_CREATOR_PERIOD_MILLIS` - The period in millis at which the StateCommitmentChainBatchCreator should attempt to create StateCommitmentChain Batches (defaults to 10,000)

#### State Commitment Chain Batch Submitter
* Postgres - See Postgres section above
* L1 Node - See L1 Node section above
  * `STATE_COMMITMENT_CHAIN_CONTRACT_ADDRESS` - (Required) The address of the StateCommitmentChain contract
* `L1_SEQUENCER_PRIVATE_KEY` - (Required) The private key to use to submit Sequencer Transaction Batches
* `FINALITY_DELAY_IN_BLOCKS` - (Required) The number of blocks required to consider a submission final on L1
* `STATE_COMMITMENT_CHAIN_BATCH_SUBMITTER_PERIOD_MILLIS` - The period in millis at which the StateCommitmentChainBatchCreator should attempt to create StateCommitmentChain Batches (defaults to 10,000)

#### Fraud Detector
* Postgres - See Postgres section above
* `FRAUD_DETECTOR_PERIOD_MILLIS` - The period in millis at which the FraudDetector should run (defaults to 10,000)
* `REALERT_ON_UNRESOLVED_FRAUD_EVERY_N_FRAUD_DETECTOR_RUNS` - The number of runs after which a detected fraud, if still present, should re-alert (via error logs) (defaults to 10)

## Building & Running
1. Make sure dependencies are installed just run `yarn` in the base directory
2. Build `yarn build`
3. Run `yarn services`

## Controlling log output verbosity
Before running, set the `DEBUG` environment variable to specify the verbosity level. It must be made up of comma-separated values of patterns to match in debug logs. Here's a few common options:
* `debug*` - Will match all debug statements -- very verbose
* `info*` - Will match all info statements -- less verbose, useful in most cases
* `warn*` - Will match all warnings -- recommended at a minimum
* `error*` - Will match all errors -- would not omit this

Examples:
* Everything but debug: `export DEBUG=info*,error*,warn*`
* Most verbose: `export DEBUG=info*,error*,warn*,debug*`
