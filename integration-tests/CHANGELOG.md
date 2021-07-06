# @eth-optimism/integration-tests

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
