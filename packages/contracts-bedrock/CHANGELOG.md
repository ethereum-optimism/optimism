# @eth-optimism/contracts-bedrock

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
