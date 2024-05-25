# Smart Contract Style Guide

This document provides guidance on how we organize and write our smart contracts. For cases where
this document does not provide guidance, please refer to existing contracts for guidance,
with priority on the `L2OutputOracle` and `OptimismPortal`.

## Standards and Conventions

### Style

#### Comments

Optimism smart contracts follow the triple-slash [solidity natspec comment style](https://docs.soliditylang.org/en/develop/natspec-format.html#documentation-example)
with additional rules. These are:

- Always use `@notice` since it has the same general effect as `@dev` but avoids confusion about when to use one over the other.
- Include a newline between `@notice` and the first `@param`.
- Include a newline between `@param` and the first `@return`.
- Use a line-length of 100 characters.

We also have the following custom tags:

- `@custom:proxied`: Add to a contract whenever it's meant to live behind a proxy.
- `@custom:upgradeable`: Add to a contract whenever it's meant to be inherited by an upgradeable contract.
- `@custom:semver`: Add to `version` variable which indicate the contracts semver.
- `@custom:legacy`: Add to an event or function when it only exists for legacy support.
- `@custom:network-specific`: Add to state variables which vary between OP Chains.

#### Errors

- Use `require` statements when making simple assertions.
- Use `revert(string)` if throwing an error where an assertion is not being made (no custom errors).
  See [here](https://github.com/ethereum-optimism/optimism/blob/861ae315a6db698a8c0adb1f8eab8311fd96be4c/packages/contracts-bedrock/contracts/L2/OVM_ETH.sol#L31)
  for an example of this in practice.
- Error strings MUST have the format `"{ContractName}: {message}"` where `message` is a lower case string.

#### Function Parameters

- Function parameters should be prefixed with an underscore.

#### Function Return Arguments

- Arguments returned by functions should be suffixed with an underscore.

#### Event Parameters

- Event parameters should NOT be prefixed with an underscore.

#### Immutable variables

Immutable variables:

- should be in `SCREAMING_SNAKE_CASE`
- should be `internal`
- should have a hand written getter function

This approach clearly indicates to the developer that the value is immutable, without exposing
the non-standard casing to the interface. It also ensure that we don’t need to break the ABIs if
we switch between values being in storage and immutable.

#### Spacers

We use spacer variables to account for old storage slots that are no longer being used.
The name of a spacer variable MUST be in the format `spacer_<slot>_<offset>_<length>` where
`<slot>` is the original storage slot number, `<offset>` is the original offset position
within the storage slot, and `<length>` is the original size of the variable.
Spacers MUST be `private`.

### Proxy by Default

All contracts should be assumed to live behind proxies (except in certain special circumstances).
This means that new contracts MUST be built under the assumption of upgradeability.
We use a minimal [`Proxy`](./contracts/universal/Proxy.sol) contract designed to be owned by a
corresponding [`ProxyAdmin`](./contracts/universal/ProxyAdmin.sol) which follow the interfaces
of OpenZeppelin's `Proxy` and `ProxyAdmin` contracts, respectively.

Unless explicitly discussed otherwise, you MUST include the following basic upgradeability
pattern for each new implementation contract:

1. Extend OpenZeppelin's `Initializable` base contract.
2. Include a `uint8 public constant VERSION = X` at the TOP of your contract.
3. Include a function `initialize` with the modifier `reinitializer(VERSION)`.
4. In the `constructor`, set any `immutable` variables and call the `initialize` function for setting mutables.

### Versioning

All (non-library and non-abstract) contracts MUST extend the `Semver` base contract which
exposes a `version()` function that returns a semver-compliant version string.

Contracts must have a `Semver` of `1.0.0` or greater to be production ready. Contracts
with `Semver` values less than `1.0.0` should only be used locally or on devnets.

Additionally, contracts MUST use the following versioning scheme:

- `patch` releases are to be used only for changes that do NOT modify contract bytecode (such as updating comments).
- `minor` releases are to be used for changes that modify bytecode OR changes that expand the contract ABI provided that these changes do NOT break the existing interface.
- `major` releases are to be used for changes that break the existing contract interface OR changes that modify the security model of a contract.

#### Exceptions

We have made an exception to the `Semver` rule for the `WETH` contract to avoid
making changes to a well-known, simple, and recognizable contract.

### Dependencies

Where basic functionality is already supported by an existing contract in the OpenZeppelin library,
we should default to using the Upgradeable version of that contract.

### Tests

Tests are written using Foundry.

All test contracts and functions should be organized and named according to the following guidelines.

These guidelines are also encoded in a script which can be run with:

```
tsx scripts/forge-test-names.ts
```

_Note: This is a work in progress, not all test files are compliant with these guidelines._

#### Organizing Principles

- Solidity `contract`s are used to organize the test suite similar to how mocha uses describe.
- Every non-trivial state changing function should have a separate contract for happy and sad path
  tests. This helps to make it very obvious where there are not yet sad path tests.
- Simpler functions like getters and setters are grouped together into test contracts.

#### Test function naming convention

Test function names are split by underscores, into 3 or 4 parts. An example function name is `test_onlyOwner_callerIsNotOwner_reverts()`.

The parts are: `[method]_[FunctionName]_[reason]_[status]`, where:

- `[method]` is either `test`, `testFuzz`, or `testDiff`
- `[FunctionName]` is the name of the function or higher level behavior being tested.
- `[reason]` is an optional description for the behavior being tested.
- `[status]` must be one of:
  - `succeeds`: used for most happy path cases
  - `reverts`: used for most sad path cases
  - `works`: used for tests which include a mix of happy and sad assertions (these should be broken up if possible)
  - `fails`: used for tests which 'fail' in some way other than reverting
  - `benchmark`: used for tests intended to establish gas costs

#### Contract Naming Conventions

Test contracts should be named one of the following according to their use:

- `TargetContract_Init` for contracts that perform basic setup to be reused in other test contracts.
- `TargetContract_Function_Test` for contracts containing happy path tests for a given function.
- `TargetContract_Function_TestFail` for contracts containing sad path tests for a given function.

To minimize clutter, getter functions can be grouped together into a single test contract,
  ie. `TargetContract_Getters_Test`.

## Withdrawing From Fee Vaults

See the file `scripts/FeeVaultWithdrawal.s.sol` to withdraw from the L2 fee vaults. It includes
instructions on how to run it. `foundry` is required.
