# L2ToL1MessagePasserTest
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L2ToL1MessagePasser.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## State Variables
### messagePasser

```solidity
L2ToL1MessagePasser messagePasser;
```


## Functions
### setUp


```solidity
function setUp() public virtual override;
```

### testFuzz_initiateWithdrawal_succeeds


```solidity
function testFuzz_initiateWithdrawal_succeeds(
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external;
```

### test_initiateWithdrawal_fromContract_succeeds


```solidity
function test_initiateWithdrawal_fromContract_succeeds() external;
```

### test_initiateWithdrawal_fromEOA_succeeds


```solidity
function test_initiateWithdrawal_fromEOA_succeeds() external;
```

### test_burn_succeeds


```solidity
function test_burn_succeeds() external;
```

## Events
### MessagePassed

```solidity
event MessagePassed(
    uint256 indexed nonce,
    address indexed sender,
    address indexed target,
    uint256 value,
    uint256 gasLimit,
    bytes data,
    bytes32 withdrawalHash
);
```

### WithdrawerBalanceBurnt

```solidity
event WithdrawerBalanceBurnt(uint256 indexed amount);
```

