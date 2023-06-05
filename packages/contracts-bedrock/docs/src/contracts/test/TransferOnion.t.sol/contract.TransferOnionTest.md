# TransferOnionTest
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/TransferOnion.t.sol)

**Inherits:**
Test

Test coverage of TransferOnion


## State Variables
### onion
TransferOnion


```solidity
TransferOnion internal onion;
```


### _token
token constructor arg


```solidity
address internal _token;
```


### _sender
sender constructor arg


```solidity
address internal _sender;
```


## Functions
### setUp

Sets up addresses, deploys contracts and funds the owner.


```solidity
function setUp() public;
```

### _deploy

Deploy the TransferOnion with a dummy shell


```solidity
function _deploy() public;
```

### _deploy

Deploy the TransferOnion with a specific shell


```solidity
function _deploy(bytes32 _shell) public;
```

### _onionize

Build the onion data


```solidity
function _onionize(TransferOnion.Layer[] memory _layers) public pure returns (bytes32, TransferOnion.Layer[] memory);
```

### test_constructor_succeeds

The constructor sets the variables as expected


```solidity
function test_constructor_succeeds() external;
```

### test_unwrap_succeeds

unwrap


```solidity
function test_unwrap_succeeds() external;
```

