# Smart Contract Style Guide

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Standards and Conventions](#standards-and-conventions)
  - [Style](#style)
    - [Comments](#comments)
    - [Errors](#errors)
    - [Function Parameters](#function-parameters)
    - [Function Return Arguments](#function-return-arguments)
    - [Event Parameters](#event-parameters)
    - [Immutable variables](#immutable-variables)
    - [Spacers](#spacers)
  - [Proxy by Default](#proxy-by-default)
  - [Versioning](#versioning)
    - [Exceptions](#exceptions)
  - [Dependencies](#dependencies)
  - [Source Code](#source-code)
  - [Tests](#tests)
    - [Expect Revert with Low Level Calls](#expect-revert-with-low-level-calls)
    - [Organizing Principles](#organizing-principles)
    - [Test function naming convention](#test-function-naming-convention)
      - [Detailed Naming Rules](#detailed-naming-rules)
    - [Contract Naming Conventions](#contract-naming-conventions)
    - [Test File Organization](#test-file-organization)
    - [Test Naming Exceptions](#test-naming-exceptions)
- [Withdrawing From Fee Vaults](#withdrawing-from-fee-vaults)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

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
the non-standard casing to the interface. It also ensures that we don’t need to break the ABIs if
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
We use a minimal [`Proxy`](../src/universal/Proxy.sol) contract designed to be owned by a
corresponding [`ProxyAdmin`](../src/universal/ProxyAdmin.sol) which follow the interfaces
of OpenZeppelin's `Proxy` and `ProxyAdmin` contracts, respectively.

Unless explicitly discussed otherwise, you MUST include the following basic upgradeability
pattern for each new implementation contract:

1. Extend OpenZeppelin's `Initializable` base contract.
2. Include a function `initialize` with the modifier `initializer()`.
3. In the `constructor`:
    1. Call `_disableInitializers()` to ensure the implementation contract cannot be initialized.
    2. Set any immutables. However, we generally prefer to not use immutables to ensure the same implementation contracts can be used for all chains, and to allow chain operators to dynamically configure parameters

Because `reinitializer(uint64 version)` is not used, the process for upgrading the implementation is to atomically:
1. Upgrade the implementation to the `StorageSetter` contract.
2. Use that to set the initialized slot (typically slot 0) to zero.
3. Upgrade the implementation to the desired new implementation and `initialize` it.

### Versioning

All (non-library and non-abstract) contracts MUST inherit the `ISemver` interface which
exposes a `version()` function that returns a semver-compliant version string.

Contracts must have a `version` of `1.0.0` or greater to be production ready.

Additionally, contracts MUST use the following versioning scheme when incrementing their version:

- `patch` releases are to be used only for changes that do NOT modify contract bytecode (such as updating comments).
- `minor` releases are to be used for changes that modify bytecode OR changes that expand the contract ABI provided that these changes do NOT break the existing interface.
- `major` releases are to be used for changes that break the existing contract interface OR changes that modify the security model of a contract.

The remainder of the contract versioning and release process can be found in [`VERSIONING.md](../policies/VERSIONING.md).

#### Exceptions

We have made an exception to the `Semver` rule for the `WETH` contract to avoid
making changes to a well-known, simple, and recognizable contract.

Additionally, bumping the patch version does change the bytecode, so another exception is carved out for this.
In other words, changing comments increments the patch version, which changes bytecode. This bytecode
change implies a minor version increment is needed, but because it's just a version change, only a
patch increment should be used.

### Dependencies

Where basic functionality is already supported by an existing contract in the OpenZeppelin library,
we should default to using the Upgradeable version of that contract.

### Source Code

The following guidelines should be followed for all contracts in the `src/` directory:

- All state changing functions should emit a corresponding event. This ensures that all actions are transparent, can be easily monitored, and can be reconstructed from the event logs.

### Tests

Tests are written using Foundry.

All test contracts and functions should be organized and named according to the following guidelines.

These guidelines are enforced by a validation script which can be run with:

```
just lint-forge-tests-check-no-build
```

The script validates both function naming conventions and contract structure requirements.

#### Expect Revert with Low Level Calls

There is a non-intuitive behavior in foundry tests, which is documented [here](https://book.getfoundry.sh/cheatcodes/expect-revert?highlight=expectrevert#expectrevert).
When testing for a revert on a low-level call, please use the `revertsAsExpected` pattern suggested there.

_Note: This is a work in progress, not all test files are compliant with these guidelines._

#### Organizing Principles

- Solidity `contract`s are used to organize the test suite similar to how mocha uses describe.
- Every function should have a separate contract for testing. This helps to make it very obvious where there are not yet tests and provides clear organization by function.

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

##### Detailed Naming Rules

Test function names must follow these strict formatting rules:

- **camelCase**: Each underscore-separated part must start with a lowercase letter
  - Valid: `test_something_succeeds`
  - Invalid: `test_Something_succeeds`
- **No double underscores**: Empty parts between underscores are not allowed
  - Valid: `test_something_succeeds`
  - Invalid: `test__something_succeeds`
- **Part count**: Must have exactly 3 or 4 parts separated by underscores
- **Failure tests**: Tests ending with `reverts` or `fails` must have 4 parts to include the failure reason
  - Valid: `test_transfer_insufficientBalance_reverts`
  - Invalid: `test_transfer_reverts`
- **Benchmark variants**:
  - Basic: `test_something_benchmark`
  - Numbered: `test_something_benchmark_123`

#### Contract Naming Conventions

Test contracts should be organized with one contract per function being tested, following these naming patterns:

- `<ContractName>_TestInit` for contracts that perform initialization/setup to be reused in other test contracts
- `<ContractName>_<FunctionName>_Test` for contracts containing tests for a specific function
- `<ContractName>_Harness` for basic harness contracts that extend functionality for testing
- `<ContractName>_<Descriptor>_Harness` for specialized harness contracts (e.g., `OPContractsManager_Upgrade_Harness`)
- `<ContractName>_Uncategorized_Test` for miscellaneous tests that don't fit specific function categories

**Legacy Notice:** The older `_TestFail` suffix is deprecated and should be updated to `_Test` with appropriate failure test naming.

#### Test File Organization

Test files must follow specific organizational requirements:

- **File location**: Test files must be placed in the `test/` directory with `.t.sol` extension
- **Source correspondence**: Each test file should have a corresponding source file in the `src/` directory
  - Test: `test/L1/OptimismPortal.t.sol`
  - Source: `src/L1/OptimismPortal.sol`
- **Name matching**: The base contract name (before first underscore) must match the filename
- **Function validation**: Function names referenced in test contract names must exist in the source contract's public interface

#### Test Naming Exceptions

Certain types of tests are excluded from standard naming conventions:

- **Invariant tests** (`test/invariants/`): Use specialized invariant testing patterns
- **Integration tests** (`test/integration/`): May test multiple contracts together
- **Script tests** (`test/scripts/`): Test deployment and utility scripts
- **Library tests** (`test/libraries/`): May have different artifact structures
- **Formal verification** (`test/kontrol/`): Use specialized tooling conventions
- **Vendor tests** (`test/vendor/`): Test external code with different patterns

## Withdrawing From Fee Vaults

See the file `scripts/FeeVaultWithdrawal.s.sol` to withdraw from the L2 fee vaults. It includes
instructions on how to run it. `foundry` is required.
