# Invariant Docs

This directory contains documentation for all defined invariant tests within `contracts-bedrock`.

<!-- Do not modify the following section manually. It will be automatically generated on running `pnpm autogen:invariant-docs` -->
<!-- START autoTOC -->

## Table of Contents
- [AddressAliasHelper](./AddressAliasHelper.md)
- [Burn.Eth](./Burn.Eth.md)
- [Burn.Gas](./Burn.Gas.md)
- [CrossDomainMessenger](./CrossDomainMessenger.md)
- [Encoding](./Encoding.md)
- [Hashing](./Hashing.md)
- [L2OutputOracle](./L2OutputOracle.md)
- [OptimismPortal](./OptimismPortal.md)
- [ResourceMetering](./ResourceMetering.md)
- [SafeCall](./SafeCall.md)
- [SystemConfig](./SystemConfig.md)
<!-- END autoTOC -->

## Usage

To auto-generate documentation for invariant tests, run `pnpm autogen:invariant-docs`.

## Documentation Standard

In order for an invariant test file to be picked up by the [docgen script](../scripts/invariant-doc-gen.ts), it must
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
