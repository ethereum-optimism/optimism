# Changelog

## 0.5.38

### Patch Changes

- Updated dependencies [1e76cdb86]
  - @eth-optimism/core-utils@0.11.0

## 0.5.37

### Patch Changes

- 628affc7: Add prefunded accounts to L2 genesis when doing local network
- 740e1bcc: Expose the deployments in the deployer image

## 0.5.36

### Patch Changes

- 7215f4ce: Bump ethers to 5.7.0 globally
- Updated dependencies [7215f4ce]
- Updated dependencies [206f6033]
  - @eth-optimism/core-utils@0.10.1

## 0.5.35

### Patch Changes

- 334a3eb0: Quick patch to fix a build issue in the contracts package

## 0.5.34

### Patch Changes

- 299157e7: Significantly reduces contracts package bundle size
- Updated dependencies [dbfea116]
  - @eth-optimism/core-utils@0.10.0

## 0.5.33

### Patch Changes

- 0c2719f8: Add inspect hh task
- a1a73e64: Updates the SDK to pull contract addresses from the deployments of the contracts package. Updates the Contracts package to export a function that makes it possible to pull deployed addresses.

## 0.5.32

### Patch Changes

- Updated dependencies [0df744f6]
- Updated dependencies [8ae39154]
- Updated dependencies [dac4a9f0]
  - @eth-optimism/core-utils@0.9.3

## 0.5.31

### Patch Changes

- 1de4f48e: Deploy goerli SCC to fix sccFaultProofWindowSeconds
- Updated dependencies [0bf3b9b4]
- Updated dependencies [8d26459b]
- Updated dependencies [4477fe9f]
  - @eth-optimism/core-utils@0.9.2

## 0.5.30

### Patch Changes

- 6e3449ba: Properly export typechain
- Updated dependencies [f9fee446]
  - @eth-optimism/core-utils@0.9.1

## 0.5.29

### Patch Changes

- Updated dependencies [700dcbb0]
  - @eth-optimism/core-utils@0.9.0

## 0.5.28

### Patch Changes

- 27234f68: Use hardhat-deploy-config for deployments
- 29ff7462: Revert es target back to 2017
- Updated dependencies [29ff7462]
  - @eth-optimism/core-utils@0.8.7

## 0.5.27

### Patch Changes

- 7c5ac36f: goerli redeploy
- 3d4d988c: package: contracts-governance

## 0.5.26

### Patch Changes

- Updated dependencies [17962ca9]
  - @eth-optimism/core-utils@0.8.6

## 0.5.25

### Patch Changes

- d18ae135: Updates all ethers versions in response to BN.js bug
- Updated dependencies [d18ae135]
  - @eth-optimism/core-utils@0.8.5

## 0.5.24

### Patch Changes

- b7a04acf: Remove unused network name parameter in contract deploy configs

## 0.5.23

### Patch Changes

- 412688d5: Replace calls to getNetwork() with getChainId util

## 0.5.22

### Patch Changes

- 51adb389: Add Teleportr mainnet deployment
- Updated dependencies [5cb3a5f7]
- Updated dependencies [6b9fc055]
  - @eth-optimism/core-utils@0.8.4

## 0.5.21

### Patch Changes

- 5818decb: Remove l2 gas price hardhat task

## 0.5.20

### Patch Changes

- d040a8d9: Deleted update and helper functions/tests from Lib_MerkleTrie.sol and Lib_SecureMerkleTrie.sol
- b57014d1: Update to typescript@4.6.2
- Updated dependencies [b57014d1]
  - @eth-optimism/core-utils@0.8.3

## 0.5.19

### Patch Changes

- c1957126: Update Dockerfile to use Alpine
- d9a51154: Bump to hardhat@2.9.1
- Updated dependencies [c1957126]
  - @eth-optimism/core-utils@0.8.2

## 0.5.18

### Patch Changes

- 88601cb7: Refactored Dockerfiles

## 0.5.17

### Patch Changes

- 175ae0bf: Minor README update

## 0.5.16

### Patch Changes

- 962f36e4: Add support for system addresses
- f2179e37: Add a fetch batches hardhat task
- b6a4fa4b: Removes outdated functions and constants from the contracts package
- b7c0a5ca: Remove yargs as a contracts dependency (unused)
- Updated dependencies [5a6f539c]
- Updated dependencies [27d8942e]
  - @eth-optimism/core-utils@0.8.1

## 0.5.15

### Patch Changes

- 78298782: Contracts are additionally verified on sourcify during deploy. This should reduce manual labor during future regeneses.
- Updated dependencies [0b4453f7]
  - @eth-optimism/core-utils@0.8.0

## 0.5.14

### Patch Changes

- Updated dependencies [b4165299]
- Updated dependencies [3c2acd91]
  - @eth-optimism/core-utils@0.7.7

## 0.5.13

### Patch Changes

- 438bc78a: Remove unused gas testing utils

## 0.5.12

### Patch Changes

- ba14c59d: Updates various ethers dependencies to their latest versions
- Updated dependencies [ba14c59d]
  - @eth-optimism/core-utils@0.7.6

## 0.5.11

### Patch Changes

- e631c39c: Add berlin hardfork config to genesis creation

## 0.5.10

### Patch Changes

- Updated dependencies [ad94b9d1]
  - @eth-optimism/core-utils@0.7.5

## 0.5.9

### Patch Changes

- Updated dependencies [ba96a455]
- Updated dependencies [c3e85fef]
  - @eth-optimism/core-utils@0.7.4

## 0.5.8

### Patch Changes

- b3efb8b7: String update to change the system name from OE to Optimism
- 279603e5: Update hardhat task for managing the gas oracle
- b6040bb3: Remove legacy bin/deploy.ts script

## 0.5.7

### Patch Changes

- b6f89fad: Adds a new TestLib_CrossDomainUtils so we can properly test cross chain encoding functions

## 0.5.6

### Patch Changes

- bbd42e03: Add config checks to validation script
- 453f0774: Copy the deployments directory into the deployer docker image

## 0.5.5

### Patch Changes

- Updated dependencies [584cbc25]
  - @eth-optimism/core-utils@0.7.3

## 0.5.4

### Patch Changes

- Updated dependencies [8e634b49]
  - @eth-optimism/core-utils@0.7.2

## 0.5.3

### Patch Changes

- b9049406: Use a gas price of zero for static calls in the deploy process
- a8b14a7d: Adds additional deploy step to transfer messenger ownership

## 0.5.2

### Patch Changes

- 243f33e5: Standardize package json file format
- Updated dependencies [243f33e5]
  - @eth-optimism/core-utils@0.7.1

## 0.5.1

### Patch Changes

- c0fc7fee: Add AddressDictator validation script

## 0.5.0

### Minor Changes

- e4a1129c: Adds aliasing to msg.sender and tx.origin to avoid xdomain attacks
- 299a459e: Introduces a new opcode L1BLOCKNUMBER to replace old functionality where blocknumber would return the L1 block number and the L2 block number was inaccessible.
- 5db50b3d: Replace the CTCs Queue storage container with a mapping
- 66bf56a6: Remove unused code from CTC
- 2c91ca00: Allow the sequencer to modify the parameters which determine the gas burn #1516
- 3f590e33: Remove the "OVM" Prefix from contract names
- e20deca0: Only burn gas in the CTC on deposits with a high gas limit
- 2a731e0d: Moves the standards folder out of libraries and into its own top-level folder.
- 872f5976: Removes various unused OVM contracts
- c53b3587: Remove queue() function from CTC.
- 1e63ffa0: Refactors and simplifies OVM_ETH usage
- b56dd079: Updates the deployment process to correctly set all constants and adds more integration tests
- 3e2aa16a: Removes the hardhat-ovm plugin and its dependencies
- d3cb1b86: Reintroduces the whitelist into the v2 system
- 973589da: Reduce CTC gas costs by storing only a blockhash.
- 81ccd6e4: `regenesis/0.5.0` release
- f38b8000: Removes ERC20 and WETH9 features from OVM_ETH
- d5f012ab: Simplify hierarchy of contracts package
- 76c84f21: Removes unused functions within OVM_BondManager
- 3605b963: Adds refactored support for the L1MESSAGESENDER opcode
- 3f28385a: Removes all custom genesis initialization
- a0947c3f: Make the CTC's gas burn amount configurable at create time

### Patch Changes

- 64ea3ac9: Run etherscan verification after each contract is deployed.
- 8c8807c0: Refactor to simplify the process of generating the genesis json file
- d7978cfc: Cleans up the contract deployment process
- e16d41c0: Correctly export contracts package types
- d5036826: Improves the build process for autogenerated files
- dfc784e8: Make the standard token factory a predeploy
- 436c48fd: Added default values library for contracts
- 2ade9a79: Removes a bunch of dead testing code in the contracts repo
- 0272a536: Adds a git commit hash to the output of make-genesis.ts
- 6ee7423f: Modifies package.json to correctly export all contracts
- 199e895e: Reduces the cost of appendSequencerBatch
- 9c1443a4: Assert upper bound on CTC gas costs
- 26906518: Always print dictator messages during deployment
- 1b917041: Add getter and setter to `OVM_GasPriceOracle` for the l1 base fee
- 483f561b: Update and harden the contract deployment process
- b70ee70c: upgraded to solidity 0.8.9
- c38e4b57: Minor bugfixes to the regenesis process for OVM_ETH
- a98a1884: Fixes dependencies instead of using caret constraints
- b744b6ea: Adds WETH9 at the old OVM_ETH address
- d2eb8ae0: Fix import bug in the state dump generation script
- ff266e9c: Reduce default gasLimit on eth deposits to the L1 Standard Bridge
- 56fe3793: Update the `deployer` docker image to build with python3
- 3e41df63: Add a more realistic CTC gas benchmark
- 9c63e9bd: Add gas benchmarks on deposits
- 280f348c: Adds an exported `names` object with contract and account names
- 51821d8f: Update the genesis file creation script to include the precompiles
- 29f1c228: Fixes a bug that made replayMessage useless.
- 8f4cb337: Removes the onlyRelayer modifier from the L1CrossDomainMessenger
- beb6c977: Remove obsoleted contract code. Improve events usability by indexing helpful params. Switch using encoded message and use decoded message from event
- 33abe73d: Add hardhat fork deploy step
- 71de86d6: Test contracts against london fork
- Updated dependencies [3ce62c81]
- Updated dependencies [cee2a464]
- Updated dependencies [222a3eef]
- Updated dependencies [896168e2]
- Updated dependencies [7c352b1e]
- Updated dependencies [b70ee70c]
- Updated dependencies [20c8969b]
- Updated dependencies [83a449c4]
- Updated dependencies [81ccd6e4]
- Updated dependencies [6d32d701]
  - @eth-optimism/core-utils@0.7.0

## 0.4.14

### Patch Changes

- 6d3e1d7f: Update dependencies
- Updated dependencies [6d3e1d7f]
- Updated dependencies [2e929aa9]
  - @eth-optimism/core-utils@0.6.1

## 0.4.13

### Patch Changes

- 7f7f35c3: contracts: remove l1 contracts from l2 state dump process
- Updated dependencies [e0be02e1]
- Updated dependencies [8da04505]
  - @eth-optimism/core-utils@0.6.0

## 0.4.12

### Patch Changes

- 468779ce: Add a getter to the ERC20 bridge interfaces, to return the address of the corresponding cross domain bridge

## 0.4.11

### Patch Changes

- 888dafca: Add etherscan verification support
- Updated dependencies [eb0854e7]
- Updated dependencies [21b17edd]
- Updated dependencies [dfe3598f]
  - @eth-optimism/core-utils@0.5.5

## 0.4.10

### Patch Changes

- 918c08ca: Bump ethers dependency to 5.4.x to support eip1559
- Updated dependencies [918c08ca]
  - @eth-optimism/core-utils@0.5.2

## 0.4.9

### Patch Changes

- ecc2f8c1: Patch so contracts package will correctly use the browser-compatible contract artifacts import

## 0.4.8

### Patch Changes

- e4fea5e0: Makes the contracts package browser compatible.

## 0.4.7

### Patch Changes

- 7f26667d: Add hardhat task for whitelisting addresses
- 77511b68: Add a hardhat task to withdraw ETH fees from L2 to L1

## 0.4.6

### Patch Changes

- 8feac092: Make it possible to override mint & burn methods in L2StandardERC20
- 4736eb2e: Add a task for setting the gas price oracle

## 0.4.5

### Patch Changes

- c73c3939: Update the typescript version to `4.3.5`
- Updated dependencies [c73c3939]
  - @eth-optimism/core-utils@0.5.1

## 0.4.4

### Patch Changes

- 063151a6: Run lint over the tasks directory

## 0.4.3

### Patch Changes

- 694cf429: Add a hardhat task for setting the L2 gas price

## 0.4.2

### Patch Changes

- 0313794b: Add a factory contract we can whitelist for the community phase which will be used by the Gateway to create standard ERC20 tokens on L2
- 21e47e1f: A small change to the L1 Messenger, which prevents an L2 to L1 call from send calling the CTC.
- Updated dependencies [049200f4]
  - @eth-optimism/core-utils@0.5.0

## 0.4.1

### Patch Changes

- 98e02cfa: Add 0.4.0 deployment artifacts

## 0.4.0

### Minor Changes

- db0dbfb2: Disables EOA contract upgrades until further notice
- 5fc728da: Add a new Standard Token Bridge, to handle deposits and withdrawals of any ERC20 token.
  For projects developing a custom bridge, if you were previously importing `iAbs_BaseCrossDomainMessenger`, you should now
  import `iOVM_CrossDomainMessenger`.
- 2e72fd90: Update AddressSet event to speed search up a bit. Breaks AddressSet API.
- e04de624: Add support for ovmCALL with nonzero ETH value

### Patch Changes

- 25f09abd: Adds ERC1271 support to default contract account
- dd8edc7b: Update the ECDSAContractAccount import path in the `contract-data.ts` file for connecting ethers contracts to the L2 contracts
- c87e4c74: Migrated from tslint to eslint. The preference for lint exceptions is as follows: line level, block level, file level, package level.
- 7f5936a8: Apply consistent styling to constants
- f87a2d00: Use dashes instead of colons in contract names
- 85da4979: Replaces RingBuffer with a simpler Buffer library
- 57ca21a2: "Adds connectL1Contracts and connectL2Contracts utility functions"
- c43b33ec: Add WETH9 compatible deposit and withdraw functions to OVM_ETH
- 26bc63ad: Deploy new Goerli contracts at d3e743aa7a406c583f7d76f4fda607f592d03e47
- a0d9e565: ECDSA account interface contract moved to predeploys dir
- 2bd49730: Deploy v0.4.0 rc to Kovan
- 38355a3b: Moved contracts in the "accounts" folder into the "predeploys" folder
- 3c2c32e1: Use predeploy constants lib for EM wrapper
- 48ece14c: Adds a temporary way to fund hardhat accounts when testing locally
- 014dea71: Removes one-off GasPriceOracle deployment file
- fa29b03e: Updates the deployment of the L1MultiMessageRelayer to NOT set the OVM_L2MessageRelayer address in the AddressManager
- 6b46c8ba: Disable upgradability from the ECDSA account instead of the EOA proxy.
- e045f582: Adds new SequencerFeeVault contract to store generated fees
- e29fab10: Token gateways pass additional information: sender and arbitrary data.
- c2a04893: Do not RLP decode the transaction in the OVM_ECDSAContractAccount
- baacda34: Introduce the L1ChugSplashProxy contract
- Updated dependencies [d9644c34]
- Updated dependencies [df5ff890]
  - @eth-optimism/core-utils@0.4.6

## 0.3.5

### Patch Changes

- 4e03f8a9: Update contracts README to add deploy instructions.
- 8e2bfd07: Introduces the congestion price oracle contract
- 245136f1: Minor change to how deploy.ts is invoked
- Updated dependencies [a64f8161]
- Updated dependencies [750a5021]
- Updated dependencies [c2b6e14b]
  - @eth-optimism/core-utils@0.4.5

## 0.3.4

### Patch Changes

- 7bf5941: Remove colon names from filenames
- Updated dependencies [f091e86]
- Updated dependencies [f880479]
  - @eth-optimism/core-utils@0.4.4

## 0.3.3

### Patch Changes

- 5e5d4a1: Separates logic for getting state dumps and making state dumps so we can bundle for browser

## 0.3.2

### Patch Changes

- 7dd2f72: Remove incorrect comment.

## 0.3.1

### Patch Changes

- 775118a: Updated package json with a missing dependency
- Updated dependencies [96a586e]
  - @eth-optimism/core-utils@0.4.3

## 0.3.0

### Minor Changes

- b799caa: Updates to use RLP encoded transactions in batches for the `v0.3.0` release

### Patch Changes

- b799caa: Add value transfer support to ECDSAContractAccount
- 6132e7a: Move various dependencies from primary deps to dev deps
- b799caa: Add ExecutionManager return data & RLP encoding
- b799caa: Makes ProxyEOA compatible with EIP1967, not backwards compatible since the storage slot changes.
- 20747fd: Set L2MessageRelayer name to L1MultiMessageRelayer when deploying to mainnet
- b799caa: Update ABI of simulateMessage to match run
- Updated dependencies [b799caa]
  - @eth-optimism/core-utils@0.4.2

## 0.2.11

### Patch Changes

- 9599b69: Fixed a bug in package json that stopped artifacts from being published

## 0.2.10

### Patch Changes

- 1d40586: Removed various unused dependencies
- 6dc1877: Heavily reduces npm package size by excluding unnecessary files.
- Updated dependencies [1d40586]
- Updated dependencies [ce7fa52]
  - @eth-optimism/core-utils@0.4.1

## 0.2.9

### Patch Changes

- d2091d4: Removed verifyExclusionProof function from MerkleTrie library.
- 0ef3069: Add pause(), blockMessage() and allowMessage() to L1 messenger
- Updated dependencies [28dc442]
- Updated dependencies [a0a0052]
  - @eth-optimism/core-utils@0.4.0

## 0.2.8

### Patch Changes

- 6daa408: update hardhat versions so that solc is resolved correctly
- ea4041b: Removed two old mock contracts
- f1f5bf2: Updates deployment files to remove colon filenames
- 9ec3ec0: Removes copies of OZ contracts in favor of importing from OZ directly
- 5f376ee: Adds config parsing to the deploy script for local deployments
- eef1df4: Minor update to package.json to correctly export typechain artifacts"
- a76cde5: Remove unused logic in ovmEXTCODECOPY
- e713cd0: Updates the `yarn build` command to not error
- 572dcbc: Add an extra event to messenger contracts to emit when a message is unsuccessfully relayed
- 6014ec0: Adds OVM_Sequencer and Deployer to the addresses.json output file
- Updated dependencies [6daa408]
- Updated dependencies [dee74ef]
- Updated dependencies [d64b66d]
  - @eth-optimism/core-utils@0.3.2

## 0.2.7

### Patch Changes

- e3f55ad: Remove trailing whitespace from many files

## 0.2.6

### Patch Changes

- ce5d596: Ports OVM_ECDSAContractAccount to use optimistic-solc.
- 1a55f64: Fix bridge contracts upgradeability by changing `Abs_L1TokenGateway.DEFAULT_FINALIZE_DEPOSIT_L2_GAS` from a storage var to an internal constant.
  Additionally, make some bridge functions virtual so they could be overriden in child contracts.
- 6e8fe1b: Removes mockOVM_ECDSAContractAccount and OVM_ProxySequencerEntrypoint, two unused contracts.
- 8d4aae4: Removed Lib_SafeExecutionManagerWrapper since it's no longer being used.
- c75a0fc: Use optimistic-solc to compile the SequencerEntrypoint. Also introduces a cache invalidation mechanism for hardhat-ovm so that we can push new compiler versions.
- d4ee2d7: Port OVM_DeployerWhitelist to use optimistic-solc.
- edb4346: Ports OVM_ProxyEOA to use optimistic-solc instead of the standard solc compiler.
- Updated dependencies [5077441]
  - @eth-optimism/core-utils@0.3.1

## 0.2.5

### Patch Changes

- Updated dependencies [91460d9]
- Updated dependencies [a0a7956]
- Updated dependencies [0497d7d]
  - @eth-optimism/core-utils@0.3.0

## 0.2.4

### Patch Changes

- 6626d99: fix contract import paths

## 0.2.3

### Patch Changes

- 5362d38: adds build files which were not published before to npm
- Updated dependencies [5362d38]
  - @eth-optimism/core-utils@0.2.1

## 0.2.2

### Patch Changes

- Updated dependencies [6cbc54d]
  - @eth-optimism/core-utils@0.2.0

## v0.1.11

- cleanup: ECDSAContractAccount
- cleanup: Proxy_EOA
- cleanup: StateManagerFactory
- cleanup: Bytes32Utils
- cleanup: Minor cleanup to state manager
- cleanup: SafetyChecker
- Remove gas estimators from gateway interface
- Add ERC1820 Registry as a precompile
- dev: Remove usage of custom concat function in Solidity
- Fix revert string generated by EM wrapper
- Update OVM_L1ERC20Gateway.sol
- Move OVM_BondManager test into the right location

## v0.1.10

Adds extensible ERC20Gateway and Improve CI.

- dev: Apply linting to all test files
- Test gas consumption of EM.run()
- Extensible deposit withdraw
- Update OVM_L2DepositedERC20.sol
- Commit state dumps to regenesis repo for new tags
- Update OVM_ChainStorageContainer.sol
- Update OVM_ECDSAContractAccount.sol
- Update OVM_CanonicalTransactionChain.sol
- Reset Context on invalid gaslimit
- [Fix] CI on merge
- [Fix] Run integration tests in forked context

## v0.1.9

Standardized ETH and ERC20 Gateways.

- Add ETH deposit contract.
- Add standard deposit/withdrawal interfaces.

## v0.1.5

Various cleanup and maintenance tasks.

- Improving comments and some names (#211)
- Add descriptive comments above the contract declaration for all 'non-abstract contracts' (#200)
- Add generic mock xdomain messenger (#209)
- Move everything over to hardhat (#208)
- Add comment to document v argument (#199)
- Add security related comments (#191)

## v0.1.4

Fix single contract redeployment & state dump script for
mainnet.

## v0.1.3

Add events to fraud proof initialization and finalization.

## v0.1.2

Npm publish integrity.

## v0.1.1

Audit fixes, deployment fixes & final parameterization.

- Add build mainnet command to package.json (#186)
- revert chain ID 422 -> 420 (#185)
- add `AddressSet` event (#184)
- Add mint & burn to L2 ETH (#178)
- Wait for deploy transactions (#180)
- Final Parameterization of Constants (#176)
- re-enable monotonicity tests (#177)
- make ovmSETNONCE notStatic (#179)
- Add reentry protection to ExecutionManager.run() (#175)
- Add nonReentrant to `relayMessage()` (#172)
- ctc: public getters, remove dead variable (#174)
- fix tainted memory bug in `Lib_BytesUtils.slice` (#171)

## v0.1.0

Initial Release
