# Invariant Docs

This directory contains documentation for all defined invariant tests within `contracts-bedrock`.

<!-- Do not modify the following section manually. It will be automatically generated on running `pnpm autogen:invariant-docs` -->
<!-- START autoTOC -->

## Table of Contents
- [AddressAliasHelper](./AddressAliasHelper.md)
- [Burn.Eth](./Burn.Eth.md)
- [Burn.Gas](./Burn.Gas.md)
- [CrossDomainMessenger](./CrossDomainMessenger.md)
- [ETHLiquidity](./ETHLiquidity.md)
- [Encoding](./Encoding.md)
- [FaultDisputeGame](./FaultDisputeGame.md)
- [Hashing](./Hashing.md)
- [InvariantTest.sol](./InvariantTest.sol.md)
- [L2OutputOracle](./L2OutputOracle.md)
- [OptimismPortal](./OptimismPortal.md)
- [OptimismPortal2](./OptimismPortal2.md)
- [OptimismSuperchainERC20](./OptimismSuperchainERC20.md)
- [ResourceMetering](./ResourceMetering.md)
- [SafeCall](./SafeCall.md)
- [SuperchainWETH](./SuperchainWETH.md)
- [SystemConfig](./SystemConfig.md)
<!-- END autoTOC -->

## Usage

To auto-generate documentation for invariant tests, run `just autogen-invariant-docs`.

## Documentation Standard

In order for an invariant test file to be picked up by the [docgen script](../scripts/autogen/generate-invariant-docs.ts), it must
adhere to the following conventions:

### Forge Invariants

All `forge` invariant tests must exist within the `contracts/test/invariants` folder, and the file name should be
`<ContractName>.t.sol`, where `<ContractName>` is the name of the contract that is being tested.

All tests within `forge` invariant files should follow the convention:

```solidity
/**
 * @custom:invariant <title>
 *
 * <longDescription>
 */
function invariant_<shortDescription>() external {
    // ...
}
```
