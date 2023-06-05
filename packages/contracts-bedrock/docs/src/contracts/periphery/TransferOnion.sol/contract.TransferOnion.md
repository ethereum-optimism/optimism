# TransferOnion
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/periphery/TransferOnion.sol)

**Inherits:**
ReentrancyGuard

TransferOnion is a hash onion for distributing tokens. The shell commits
to an ordered list of the token transfers and can be permissionlessly
unwrapped in order. The SENDER must `approve` this contract as
`transferFrom` is used to move the token balances.


## State Variables
### TOKEN
Address of the token to distribute.


```solidity
ERC20 public immutable TOKEN;
```


### SENDER
Address of the account to distribute tokens from.


```solidity
address public immutable SENDER;
```


### shell
Current shell hash.


```solidity
bytes32 public shell;
```


## Functions
### constructor


```solidity
constructor(ERC20 _token, address _sender, bytes32 _shell);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_token`|`ERC20`| Address of the token to distribute.|
|`_sender`|`address`|Address of the sender to distribute from.|
|`_shell`|`bytes32`| Initial shell of the onion.|


### peel

Peels layers from the onion and distributes tokens.


```solidity
function peel(Layer[] memory _layers) public nonReentrant;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_layers`|`Layer[]`|Array of onion layers to peel.|


## Structs
### Layer
Struct representing a layer of the onion.


```solidity
struct Layer {
    address recipient;
    uint256 amount;
    bytes32 shell;
}
```

