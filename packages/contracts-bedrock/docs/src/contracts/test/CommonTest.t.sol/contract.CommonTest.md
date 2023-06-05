# CommonTest
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CommonTest.t.sol)

**Inherits:**
Test


## State Variables
### alice

```solidity
address alice = address(128);
```


### bob

```solidity
address bob = address(256);
```


### multisig

```solidity
address multisig = address(512);
```


### ZERO_ADDRESS

```solidity
address immutable ZERO_ADDRESS = address(0);
```


### NON_ZERO_ADDRESS

```solidity
address immutable NON_ZERO_ADDRESS = address(1);
```


### NON_ZERO_VALUE

```solidity
uint256 immutable NON_ZERO_VALUE = 100;
```


### ZERO_VALUE

```solidity
uint256 immutable ZERO_VALUE = 0;
```


### NON_ZERO_GASLIMIT

```solidity
uint64 immutable NON_ZERO_GASLIMIT = 50000;
```


### nonZeroHash

```solidity
bytes32 nonZeroHash = keccak256(abi.encode("NON_ZERO"));
```


### NON_ZERO_DATA

```solidity
bytes NON_ZERO_DATA = hex"0000111122223333444455556666777788889999aaaabbbbccccddddeeeeffff0000";
```


### ffi

```solidity
FFIInterface ffi;
```


## Functions
### setUp


```solidity
function setUp() public virtual;
```

### emitTransactionDeposited


```solidity
function emitTransactionDeposited(
    address _from,
    address _to,
    uint256 _mint,
    uint256 _value,
    uint64 _gasLimit,
    bool _isCreation,
    bytes memory _data
) internal;
```

## Events
### TransactionDeposited

```solidity
event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData);
```

