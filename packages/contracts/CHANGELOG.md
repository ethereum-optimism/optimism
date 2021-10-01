# Changelog

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
