# @eth-optimism/integration-tests

## 0.5.22

### Patch Changes

- 1d3c749a2: Bumps the version of ts-node used
- 1d3c749a2: Updates the version of TypeScript

## 0.5.21

### Patch Changes

- a3242d4f: Fix erc721 factory to match erc21 factory

## 0.5.20

### Patch Changes

- 02c457a5: Removes NFT refund logic if withdrawals fail.

## 0.5.19

### Patch Changes

- 5c3f2b1f: Fixes NFT bridge related contracts in response to the OpenZeppelin audit. Updates tests to support these changes, including integration tests.

## 0.5.18

### Patch Changes

- 7215f4ce: Bump ethers to 5.7.0 globally

## 0.5.17

### Patch Changes

- d97df13a: Modularize the itests away from depending on api of messenger

## 0.5.16

### Patch Changes

- 977493bc: Update SDK version and usage to account for new constructor

## 0.5.15

### Patch Changes

- 29ff7462: Revert es target back to 2017

## 0.5.14

### Patch Changes

- f688a631: integration-tests: Override default bridge adapters
- d18ae135: Updates all ethers versions in response to BN.js bug

## 0.5.13

### Patch Changes

- 412688d5: Replace calls to getNetwork() with getChainId util

## 0.5.12

### Patch Changes

- 53fac1df: Facilitate actor testing on nightly

## 0.5.11

### Patch Changes

- 36a91c30: Fix various actor tests

## 0.5.10

### Patch Changes

- db02f97f: Add tests for system addrs on verifiers/replicas

## 0.5.9

### Patch Changes

- 5bf390b4: Update chainid
- c1957126: Update Dockerfile to use Alpine
- d9a51154: Bump to hardhat@2.9.1

## 0.5.8

### Patch Changes

- 88807f03: Add integration test for healthcheck server

## 0.5.7

### Patch Changes

- 88601cb7: Refactored Dockerfiles

## 0.5.6

### Patch Changes

- 962f36e4: Add support for system addresses
- d6e309be: Add test coverage for zlib compressed batches
- 386df4dc: Replaces contract references in integration tests with SDK CrossChainMessenger objects.

## 0.5.5

### Patch Changes

- 45642dc8: Replaces l1Provider and l2Provider with env.l1Provider and env.l2Provider respectively.

## 0.5.4

### Patch Changes

- dc5f6517: Deletes watcher-utils.ts. Moves it's utilities into env.ts.
- dcdcc757: Removes message relaying utilities from the Message Relayer, to be replaced by the SDK

## 0.5.3

### Patch Changes

- a8a74a98: Remove Watcher usage from itests
- e2ad8653: Support non-well-known networks
- 152df378: Use new asL2Provider function for integration tests
- 748c04ab: Updates integration tests to use the SDK for bridged token tests
- 8cb2535b: Skip an unreliable test

## 0.5.2

### Patch Changes

- d6c2830a: Increase withdrawal test timeout
- 0293749e: Add an integration test showing the infeasability of withdrawing a fake token in exchange for a legitimate token.
- a135aa3d: Updates integration tests to include a test for syncing a Verifier from L1
- 0bb11484: Remove nightly itests - not needed anymore
- ba14c59d: Updates various ethers dependencies to their latest versions
- a135aa3d: Add verifier integration tests
- edb21845: Updates integration tests to start using SDK

## 0.5.1

### Patch Changes

- e631c39c: Add in berlin hardfork tests

## 0.5.0

### Minor Changes

- c1e923f9: Updates to work with a live network

### Patch Changes

- 968fb38d: Use hardhat-ethers for importing factories in integration tests
- a7fbafa8: Split OVMMulticall.sol into Multicall.sol & OVMContext.sol

## 0.4.2

### Patch Changes

- 5787a55b: Updates to support nightly actor tests
- dad6fd9b: Update timestamp assertion for new logic

## 0.4.1

### Patch Changes

- a8013127: Remove sync-tests as coverage lives in itests now
- b1fa3f33: Enforce fees in docker-compose setup and test cases for fee too low and fee too high
- 4559a824: Pass through starting block height to dtl

## 0.4.0

### Minor Changes

- 3ce64804: Add actor tests

## 0.3.3

### Patch Changes

- 0ab37fc9: Update to node.js version 16

## 0.3.2

### Patch Changes

- d141095c: Allow for unprotected transactions

## 0.3.1

### Patch Changes

- 243f33e5: Standardize package json file format

## 0.3.0

### Minor Changes

- e03dcead: Start refactor to new version of the OVM
- e4a1129c: Adds aliasing to msg.sender and tx.origin to avoid xdomain attacks
- 3f590e33: Remove the "OVM" Prefix from contract names
- 872f5976: Removes various unused OVM contracts
- 92c9692d: Opcode tweaks. Coinbase returns SequencerFeeVault address. Difficulty returns zero.
- 1e63ffa0: Refactors and simplifies OVM_ETH usage
- b56dd079: Updates the deployment process to correctly set all constants and adds more integration tests
- 81ccd6e4: `regenesis/0.5.0` release
- f38b8000: Removes ERC20 and WETH9 features from OVM_ETH
- 3605b963: Adds refactored support for the L1MESSAGESENDER opcode

### Patch Changes

- 299a459e: Introduces a new opcode L1BLOCKNUMBER to replace old functionality where blocknumber would return the L1 block number and the L2 block number was inaccessible.
- 343da72a: Add tests for optimistic ethereum related fields to the receipt
- 7b761af5: Add updated fee scheme integration tests
- b70ee70c: upgraded to solidity 0.8.9
- a98a1884: Fixes dependencies instead of using caret constraints

## 0.2.4

### Patch Changes

- 6d3e1d7f: Update dependencies

## 0.2.3

### Patch Changes

- 918c08ca: Bump ethers dependency to 5.4.x to support eip1559

## 0.2.2

### Patch Changes

- c73c3939: Update the typescript version to `4.3.5`

## 0.2.1

### Patch Changes

- f1dc8b77: Add various stress tests

## 0.2.0

### Minor Changes

- aa6fad84: Various updates to integration tests so that they can be executed against production networks

## 0.1.2

### Patch Changes

- b107a032: Make expectApprox more readable by passing optional args as an object with well named keys

## 0.1.1

### Patch Changes

- 40b99a6e: Add new RPC endpoint `rollup_gasPrices`

## 0.1.0

### Minor Changes

- e04de624: Add support for ovmCALL with nonzero ETH value

### Patch Changes

- 25f09abd: Adds ERC1271 support to default contract account
- 5fc728da: Add a new Standard Token Bridge, to handle deposits and withdrawals of any ERC20 token.
  For projects developing a custom bridge, if you were previously importing `iAbs_BaseCrossDomainMessenger`, you should now
  import `iOVM_CrossDomainMessenger`.
- c43b33ec: Add WETH9 compatible deposit and withdraw functions to OVM_ETH
- e045f582: Adds new SequencerFeeVault contract to store generated fees
- b8e2d685: Add replica sync test to integration tests; handle 0 L2 blocks in DTL

## 0.0.7

### Patch Changes

- d1680052: Reduce test timeout from 100 to 20 seconds
- c2b6e14b: Implement the latest fee spec such that the L2 gas limit is scaled and the tx.gasPrice/tx.gasLimit show correctly in metamask
- 77108d37: Add verifier sync test and extra docker-compose functions

## 0.0.6

### Patch Changes

- f091e86: Fix to ensure that L1 => L2 success status is reflected correctly in receipts
- f880479: End to end fee integration with recoverable L2 gas limit

## 0.0.5

### Patch Changes

- 467d6cb: Adds a test for contract deployments that run out of gas

## 0.0.4

### Patch Changes

- b799caa: Add support for parsed revert reasons in DoEstimateGas
- b799caa: Update minimum response from estimate gas
- b799caa: Add value transfer support to ECDSAContractAccount
- b799caa: Update expected gas prices based on minimum of 21k value

## 0.0.3

### Patch Changes

- 6daa408: update hardhat versions so that solc is resolved correctly
- 5b9be2e: Correctly set the OVM context based on the L1 values during `eth_call`. This will also set it during `eth_estimateGas`. Add tests for this in the integration tests

## 0.0.2

### Patch Changes

- 6bcf22b: Add contracts for OVM context test coverage and add tests
