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

