# integration_tests/bedrock_test.go

Integration tests for the go service for bedrock

- **Overview**

The specific components involved in this test are 
- the Ethereum smart contracts
- the Ethereum network clients
- the Optimism L1 and L2 chains
- the Ethereum-Optimism indexers
- Postgresql database

#### Test summary

The key aspect being tested here is the correct operation of the Ethereum-Optimism indexer, specifically its ability to correctly index deposit and withdrawal transactions on both the L1 and L2 chains. 

The Ethereum clients and the Ethereum-Optimism bindings are used to create and process the transactions, while the database is used by the indexer to store the indexed data.

The test has the following sections:

- **Test Setup and cleanup** 

The test initializes a test database and a system based on the Ethereum-Optimism platform, including L1 and L2 Ethereum clients.
The test ends by cleaning up the system and the test database.

- **Ethereum Client Initialization**

The Ethereum clients are initialized using the Ethereum-Optimism bindings.

- **Transaction Creation** 

A transaction to deposit ETH into the system is created and broadcast to the L1 chain. The test then waits for the transaction to be processed and included in a block.

- **Indexer Polling** 

The test then checks the indexer to confirm that the deposit transaction has been properly indexed. This is done by polling the indexer's REST API until it returns the expected result.

- **Withdrawal Test** 

The test creates another transaction to withdraw half of the deposited ETH from the system through the L2 chain. It then waits for this transaction to be processed and checks the indexer again to confirm that the withdrawal transaction has been indexed.

- **Finalization of Withdrawal** 

The test waits for the withdrawal transaction to be finalized on the L1 chain. It then checks the indexer again to confirm that the finalization has been properly indexed.

