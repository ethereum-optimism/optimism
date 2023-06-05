# Semver_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Semver.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)

Test the Semver contract that is used for semantic versioning
of various contracts.


## State Variables
### semver
Global semver contract deployed in setUp. This is used in
the test cases.


```solidity
Semver semver;
```


## Functions
### setUp

Deploy a Semver contract


```solidity
function setUp() public virtual override;
```

### test_version_succeeds

Test the version getter


```solidity
function test_version_succeeds() external;
```

### test_behindProxy_succeeds

Since the versions are all immutable, they should
be able to be accessed from behind a proxy without needing
to initialize the contract.


```solidity
function test_behindProxy_succeeds() external;
```

