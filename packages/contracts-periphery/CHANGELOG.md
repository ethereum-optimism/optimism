# @eth-optimism/contracts-periphery

## 0.2.0

### Minor Changes

- 8a335b7b: Fixes a bug in the OptimismMintableERC721. Requires an interface change, so this is a minor and not patch.

### Patch Changes

- 95fc3fbf: Add typechain with ethers v5 support
- 019657db: Add TeleportrDeposit and TeleportrDisburser to contracts-periphery
- 6ff5c0a3: Cleaned up natspec for Drippie and its dependencies
- 119f0e97: Moves TeleportrWithdrawer to L1 contracts folder
- 9c8b1f00: Bump forge-std to 62caef29b0f87a2c6aaaf634b2ca4c09b6867c92
- 89d01f2e: Update dev deps

## 0.1.5

### Patch Changes

- 3799bb6f: Deploy Drippie to mainnet

## 0.1.4

### Patch Changes

- 9aa8049c: Deploy NFT bridge contracts

## 0.1.3

### Patch Changes

- da1633a3: ERC721 bridge from Eth Mainnet to Optimism
- 61a30273: Simplify, cleanup, and standardize ERC721 bridge contracts.
- a320e744: Updates contracts-periphery to use the standardized hardhat deploy config plugin
- 29ff7462: Revert es target back to 2017
- 604dd315: Deploy Drippie to kovan and OP kovan

## 0.1.2

### Patch Changes

- e0b89fcd: Re-deploy RetroReceiver
- 982cb980: Tweaks Drippie contract for client-side ease
- 9142adc4: Adds new TeleportrWithdrawer contract for withdrawing from Teleportr

## 0.1.1

### Patch Changes

- 416d2e60: Introduce the Drippie peripheral contract for managing ETH drips

## 0.1.0

### Minor Changes

- f7d964d7: Releases the first version of the contracts-periphery package

### Patch Changes

- d18ae135: Updates all ethers versions in response to BN.js bug
- Updated dependencies [d18ae135]
  - @eth-optimism/core-utils@0.8.5
