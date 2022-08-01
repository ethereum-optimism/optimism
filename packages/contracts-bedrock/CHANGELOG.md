# @eth-optimism/contracts-bedrock

## 0.5.2

### Patch Changes

- 1a22e822: Standardizes revert strings globally
- 5e113137: Fixes a bug in the L2 Bedrock genesis script
- 177a9ea8: Cleans linting errors in MerkleTrie.sol
- 7d68f82f: Adds a new event SentMessageExtension1 to the CrossDomainMessenger contract. Includes additional data that's being attached to messages sent after the Bedrock upgrade.
- 90630336: Properly generates and exports ABI and artifact files that can be imported by client libraries
- 8bd7abde: Moves various legacy contracts into the legacy folder
- 7e6eb9b2: The output oracle's getL2Output function now reverts when no output is returned
- f243dacf: Bump to use solidity 0.8.15
- 8d26459b: Remove subversion byte from deposit tx
- fa9823f3: Naming improvements for functions and variables in the L2OutputOracle
- 0bf3b9b4: Update forge-std
- e764cbb7: Shortens library names
- 3a0271f8: Introduces Types.sol
- 5de373ea: Semver contract updated to include a getter for the full version string
- Updated dependencies [0bf3b9b4]
- Updated dependencies [8d26459b]
- Updated dependencies [4477fe9f]
  - @eth-optimism/core-utils@0.9.2

## 0.5.1

### Patch Changes

- e4693481: Clean up BytesUtils
- b7b77d6c: Updates CrossDomainMessenger.baseGas to more accurately reflect gas costs
- 9d435aec: Cleans up natspec in MerkleTrie and SecureMerkleTrie contracts
- 87f745b5: Cleans up various compiler warnings
- 8a3074ab: Minor cleanups to initialization and semver for L1 contracts
- e1501bc0: Clears most contract linting warnings

## 0.5.0

### Minor Changes

- 42a4cc30: Remove Lib* and OVM* prefixes from all contracts

### Patch Changes

- 0cb3929e: Move encoding and hashing into Encoding and Hashing libraries
- 28bd76ae: Cleans up hashing and encoding library natspec and function names
- 4279647f: Port RLPWriter tests
- ce6cb121: Use external version of ExcessivelySafeCall
- 8986f165: Fix solc warnings in ProxyAdmin
- 69ee689f: Remove unnecessary DefaultValues library
- 2e89f634: Fixes a bug that caused L2 timestamps to be computed incorrectly
- 49d33b08: Standardizes comments, errors, and events for contracts in the /universal package
- 821907e2: Bump typechain to 8.1.0
- 91b31168: Clean up comments and errors for legacy contracts
- 3c5726d4: Cleaned up enums, should be CapitalCase enums and UPPER_CASE values
- eb11a5bb: Add comments to RLP libraries
- 092b0901: Update to new L2 tx hash style for deposits
- 4ea33e13: Standardizes initialization logic for L1 contracts
- 297af083: Move contracts written by external parties into a vendor folder
- 71800503: Reduce the number of compiler warnings
- 611d93a1: Remove storage slot buffer in xdomain messengers
- 75089d0a: Cleans up initialization logic everywhere
- b9a90f32: Rename OptimismMintableTokenFactory to OptimismMintableERC20Factory
- 50e20ea1: Fix initialization logic
- 6f74ca9f: Clean up the PredeployAddresses library
- c031ec95: Tests for RLPReader
- 9c8b1f00: Bump forge-std to 62caef29b0f87a2c6aaaf634b2ca4c09b6867c92
- 89d01f2e: Add semver to L2 contracts
- 7d9820b6: Resolve compiler warnings in Proxy.sol
- f9fee446: Move the `DepositTx` type to `core-utils`. This way it can be more easily used across projects
- 5050e0fb: Remove "not implemented" errors in virtual functions
- 78d7c2ec: Update typechain pipeline
- 89d01f2e: Update dev deps
- Updated dependencies [f9fee446]
  - @eth-optimism/core-utils@0.9.1

## 0.4.1

### Patch Changes

- 5c3b4bfa: Enable hardhat style buildinfo
- ef29d8a5: Make the Portal upgradeable
- 5bb6f2c7: Add `OptimismPortal.isOutputFinalized`
- 79f31007: correct l33t sp34k in toCodeAddrr
- 5a12c635: Add deployer docker image
- 8460865f: Optimize buildinfo support, only build through hardhat interface

## 0.4.0

### Minor Changes

- a828da9f: Add separate sequencer role to Oracle

### Patch Changes

- a828da9f: Separate the owner and sequencer roles in the OutputOracle
- 347fd37c: Fix bug in bedrock deploy scripts
- 700dcbb0: Add genesis script
- 931e517b: Fix order of args to L2OO constructor
- 93e2f750: Fix for incorrect constructor args in deploy config
- ddf515cb: Make the output oracle upgradeable.
- Updated dependencies [700dcbb0]
  - @eth-optimism/core-utils@0.9.0

## 0.3.0

### Minor Changes

- 35757456: Replaces L2 timestamps with block numbers as the key in mapping(uint => OutputProposal).

### Patch Changes

- f23bae0b: bedrock: ProxyAdmin rename OpenZeppelin proxy to ERC1967
- fadb1a93: OZ Audit fixes with a Low or informational severity:

  - Hardcode constant values
  - Require that msg.value == \_amount on ETH withdrawals
  - use \_from in place of msg.sender when applicable in internal functions

- f23bae0b: bedrock: Simplify ProxyAdmin static calls
- 650ca6d4: Fixes to medium severity OZ findings

  - Disallow reentrant withdrawals
  - remove donateEth
  - Correct ordering of \_from and \_to arguments on refunds of failed deposits

- 9aa8049c: Have contracts-bedrock properly include contract sources in npm package

## 0.2.0

### Minor Changes

- 04884132: Corrects the ordering of token addresses when a finalizeBridgeERC20 call fails

### Patch Changes

- 0a5ca8bf: Deployment for bedrock contracts on goerli
- 2f3fae0e: Fix hh artifact schema
- a96cbe7c: Fix style for L2 contracts to match L1 contracts
- 29ff7462: Revert es target back to 2017
- 14dd80f3: Add proxy contract
- Updated dependencies [29ff7462]
  - @eth-optimism/core-utils@0.8.7

## 0.1.3

### Patch Changes

- c258acd4: Update comments and style for L1 contracts

## 0.1.2

### Patch Changes

- 07a84aed: Move core-utils to deps instead of devdeps

## 0.1.1

### Patch Changes

- 1aca58c4: Initial release
