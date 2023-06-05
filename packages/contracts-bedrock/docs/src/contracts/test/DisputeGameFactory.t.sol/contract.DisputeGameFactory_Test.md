# DisputeGameFactory_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/DisputeGameFactory.t.sol)

**Inherits:**
Test


## State Variables
### factory

```solidity
DisputeGameFactory factory;
```


### fakeClone

```solidity
FakeClone fakeClone;
```


## Functions
### setUp


```solidity
function setUp() public;
```

### testFuzz_create_succeeds

*Tests that the `create` function succeeds when creating a new dispute game
with a `GameType` that has an implementation set.*


```solidity
function testFuzz_create_succeeds(uint8 gameType, Claim rootClaim, bytes calldata extraData) public;
```

### testFuzz_create_noImpl_reverts

*Tests that the `create` function reverts when there is no implementation
set for the given `GameType`.*


```solidity
function testFuzz_create_noImpl_reverts(uint8 gameType, Claim rootClaim, bytes calldata extraData) public;
```

### testFuzz_create_sameUUID_reverts

*Tests that the `create` function reverts when there exists a dispute game with the same UUID.*


```solidity
function testFuzz_create_sameUUID_reverts(uint8 gameType, Claim rootClaim, bytes calldata extraData) public;
```

### test_setImplementation_succeeds

*Tests that the `setImplementation` function properly sets the implementation for a given `GameType`.*


```solidity
function test_setImplementation_succeeds() public;
```

### test_setImplementation_notOwner_reverts

*Tests that the `setImplementation` function reverts when called by a non-owner.*


```solidity
function test_setImplementation_notOwner_reverts() public;
```

### testDiff_getGameUUID_succeeds

*Tests that the `getGameUUID` function returns the correct hash when comparing
against the keccak256 hash of the abi-encoded parameters.*


```solidity
function testDiff_getGameUUID_succeeds(uint8 gameType, Claim rootClaim, bytes calldata extraData) public;
```

### test_owner_succeeds

*Tests that the `owner` function returns the correct address after deployment.*


```solidity
function test_owner_succeeds() public;
```

### test_transferOwnership_succeeds

*Tests that the `transferOwnership` function succeeds when called by the owner.*


```solidity
function test_transferOwnership_succeeds() public;
```

### test_transferOwnership_notOwner_reverts

*Tests that the `transferOwnership` function reverts when called by a non-owner.*


```solidity
function test_transferOwnership_notOwner_reverts() public;
```

## Events
### DisputeGameCreated

```solidity
event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);
```

### ImplementationSet

```solidity
event ImplementationSet(address indexed impl, GameType indexed gameType);
```

