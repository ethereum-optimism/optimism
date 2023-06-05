# BondManager_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/BondManager.t.sol)

**Inherits:**
Test


## State Variables
### factory

```solidity
DisputeGameFactory factory;
```


### bm

```solidity
BondManager bm;
```


## Functions
### setUp


```solidity
function setUp() public;
```

### testFuzz_post_succeeds

-------------------------------------------
Test Bond Posting
-------------------------------------------

Tests that posting a bond succeeds.


```solidity
function testFuzz_post_succeeds(bytes32 bondId, address owner, uint256 minClaimHold, uint256 amount) public;
```

### testFuzz_post_duplicates_reverts

Tests that posting a bond with the same id twice reverts.


```solidity
function testFuzz_post_duplicates_reverts(bytes32 bondId, address owner, uint256 minClaimHold, uint256 amount) public;
```

### testFuzz_post_zeroAddress_reverts

Posting with the zero address as the owner fails.


```solidity
function testFuzz_post_zeroAddress_reverts(bytes32 bondId, uint256 minClaimHold, uint256 amount) public;
```

### testFuzz_post_zeroAddress_reverts

Posting zero value bonds should revert.


```solidity
function testFuzz_post_zeroAddress_reverts(bytes32 bondId, address owner, uint256 minClaimHold) public;
```

### testFuzz_seize_missingBond_reverts

-------------------------------------------
Test Bond Seizing
-------------------------------------------

Non-existing bonds shouldn't be seizable.


```solidity
function testFuzz_seize_missingBond_reverts(bytes32 bondId) public;
```

### testFuzz_seize_expired_reverts

Bonds that expired cannot be seized.


```solidity
function testFuzz_seize_expired_reverts(bytes32 bondId, address owner, uint256 minClaimHold, uint256 amount) public;
```

### testFuzz_seize_unauthorized_reverts

Bonds cannot be seized by unauthorized parties.


```solidity
function testFuzz_seize_unauthorized_reverts(bytes32 bondId, address owner, uint256 minClaimHold, uint256 amount)
    public;
```

### testFuzz_seize_succeeds

Seizing a bond should succeed if the game resolves.


```solidity
function testFuzz_seize_succeeds(bytes32 bondId, uint256 minClaimHold, bytes calldata extraData) public;
```

### testFuzz_seizeAndSplit_succeeds

-------------------------------------------
Test Bond Split and Seizing
-------------------------------------------

Seizing and splitting a bond should succeed if the game resolves.


```solidity
function testFuzz_seizeAndSplit_succeeds(bytes32 bondId, uint256 minClaimHold, bytes calldata extraData) public;
```

### testFuzz_reclaim_succeeds

-------------------------------------------
Test Bond Reclaiming
-------------------------------------------

Bonds can be reclaimed after the specified amount of time.


```solidity
function testFuzz_reclaim_succeeds(bytes32 bondId, address owner, uint256 minClaimHold, uint256 amount) public;
```

## Events
### DisputeGameCreated

```solidity
event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);
```

### BondPosted

```solidity
event BondPosted(bytes32 bondId, address owner, uint256 expiration, uint256 amount);
```

### BondSeized

```solidity
event BondSeized(bytes32 bondId, address owner, address seizer, uint256 amount);
```

### BondReclaimed

```solidity
event BondReclaimed(bytes32 bondId, address claiment, uint256 amount);
```

