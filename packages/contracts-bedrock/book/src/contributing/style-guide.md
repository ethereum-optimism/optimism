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

This document provides guidance on how we organize and write our smart contracts.

Notes:
1. There are many cases where the code is not up to date with this guide, when in doubt, this guide
   should take precedence.
2. For cases where this document does not provide guidance, please refer to existing contracts,
   with priority on the `SystemConfig` and `OptimismPortal`.

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

- Prefer custom Solidity errors for all new errors.
- Name custom errors using `ContractName_ErrorDescription`.
- Use `revert ContractName_ErrorDescription()` to revert.
- Avoid `revert(string)` and string-typed error messages in new code.

Example:

```solidity
// ✅ Correct - Custom errors with contract-prefixed names
contract SystemConfig {
    error SystemConfig_InvalidFeatureState();
    error SystemConfig_UnauthorizedCaller(address caller);

    address internal owner;

    function setFeature(bool _enabled) external {
        if (msg.sender != owner) revert SystemConfig_UnauthorizedCaller(msg.sender);
        if (!_enabled) revert SystemConfig_InvalidFeatureState();
        // ...
    }
}

// ❌ Incorrect - string-based reverts and contract-prefixed strings
function bad(uint256 _amount) external {
    require(_amount > 0, "MyContract: amount must be > 0"); // Prefer custom error
    revert("MyContract: unsupported"); // Avoid string reverts
}
```

#### Function Parameters

- Function parameters should be prefixed with an underscore.

Example:

```solidity
// ✅ Correct - parameters are prefixed with underscore
function setOwner(address _newOwner) external {
    // ...
}

// ❌ Incorrect - parameters without underscore prefix
function setOwner(address newOwner) external {
    // ...
}
```

#### Function Return Arguments

- Arguments returned by functions should be suffixed with an underscore.

Example:

```solidity
// ✅ Correct - return variable is suffixed with underscore
function balanceOf(address _account) public view returns (uint256 balance_) {
    balance_ = balances[_account];
}

// ❌ Incorrect - return variable without underscore suffix
function balanceOf(address _account) public view returns (uint256 balance) {
    balance = balances[_account];
}
```

#### Event Parameters

- Event parameters should be named using camelCase.
- Event parameters should NOT be prefixed with an underscore.

Example:

```solidity
// ✅ Correct - event params are not prefixed with underscore
event OwnerChanged(address previousOwner, address newOwner);

// ❌ Incorrect - event params prefixed with underscore
event OwnerChanged(address _previousOwner, address _newOwner);

// ❌ Incorrect - event params are not camelCase or are unnamed
event OwnerChanged(address, address NEW_OWNER);

```

#### Immutable variables

Immutable variables:

- should be in `SCREAMING_SNAKE_CASE`
- should be `internal`
- should have a hand written getter function

This approach clearly indicates to the developer that the value is immutable, without exposing
the non-standard casing to the interface. It also ensures that we don’t need to break the ABIs if
we switch between values being in storage and immutable.

Example:

```solidity
contract ExampleWithImmutable {
    // ❌ Incorrect - immutable is not SCREAMING_SNAKE_CASE
    address internal immutable ownerAddress;

    // ❌ Incorrect - immutable is public
    address public immutable ownerAddress;

    // ✅ Correct - immutable is internal and SCREAMING_SNAKE_CASE
    address internal immutable OWNER_ADDRESS;

    constructor(address _owner) {
        OWNER_ADDRESS = _owner;
    }

    // ✅ Handwritten getter
    function ownerAddress() public view returns (address) {
        return OWNER_ADDRESS;
    }
}
```

#### Spacers

We use spacer variables to account for old storage slots that are no longer being used.
The name of a spacer variable MUST be in the format `spacer_<slot>_<offset>_<length>` where
`<slot>` is the original storage slot number, `<offset>` is the original offset position
within the storage slot, and `<length>` is the original size of the variable.
Spacers MUST be `private`.

Example:

```solidity
contract ExampleStorageV2 {
    // ✅ Correct - spacer preserves old storage layout
    bytes32 private spacer_5_0_32;
    uint256 public value;
}

// ❌ Incorrect - wrong visibility and/or naming
contract BadStorageLayout {
    bytes32 internal spacer5;
}
```

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

### Interface Inheritance

In order to reduce build times, all external dependencies (ie. a contract that is being interacted with)
should be imported as interfaces. In order to facilitate this, implementation contracts must have an
associated interface in the `interfaces/` directory of the contracts package. Checks in CI
will ensure that the interface exists and is correct. These interfaces should include a
"pseudo-constructor" function (`function __constructor__()`) which ensures that the constructor's
encoding is exposed in the ABI.

Contracts must not inherit from their own interfaces (e.g., `contract SomeContract is ISomeContract`).
Interfaces may or may not inherit from other interfaces to compose functionality.

**Rationale:**

- **Alignment Issues**: If a contracts inherits from a base contracts (like `Ownable`), it cannot inherit from the interface as well, as this prevents 1:1 alignment between the implementation and interface, since the interface cannot include the base contract functions (ie. `owner()`) without causing compiler errors.
- **Constructor Complications**: Interface inheritance can cause issues with pseudo-constructors.

**Example:**

```solidity
// ✅ Correct - contract inherits from base contracts, interface composes other interfaces
contract SomeContract is SomeBaseContract, ... {
    // Implementation
}

interface ISomeContract is ISomeBaseContract {
    // Interface definition
}

// ❌ Incorrect - contract inheriting from its own interface
contract SomeContract is ISomeContract, ... {
    // This creates alignment and compilation issues
}
```

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
