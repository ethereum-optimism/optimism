# OptimismMintableERC20
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/OptimismMintableERC20.sol)

**Inherits:**
[IOptimismMintableERC20](/contracts/universal/IOptimismMintableERC20.sol/interface.IOptimismMintableERC20.md), [ILegacyMintableERC20](/contracts/universal/IOptimismMintableERC20.sol/interface.ILegacyMintableERC20.md), ERC20, [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

OptimismMintableERC20 is a standard extension of the base ERC20 token contract designed
to allow the StandardBridge contracts to mint and burn tokens. This makes it possible to
use an OptimismMintablERC20 as the L2 representation of an L1 token, or vice-versa.
Designed to be backwards compatible with the older StandardL2ERC20 token which was only
meant for use on L2.


## State Variables
### REMOTE_TOKEN
Address of the corresponding version of this token on the remote chain.


```solidity
address public immutable REMOTE_TOKEN;
```


### BRIDGE
Address of the StandardBridge on this network.


```solidity
address public immutable BRIDGE;
```


## Functions
### onlyBridge

A modifier that only allows the bridge to call


```solidity
modifier onlyBridge();
```

### constructor


```solidity
constructor(address _bridge, address _remoteToken, string memory _name, string memory _symbol)
    ERC20(_name, _symbol)
    Semver(1, 0, 0);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bridge`|`address`|     Address of the L2 standard bridge.|
|`_remoteToken`|`address`|Address of the corresponding L1 token.|
|`_name`|`string`|       ERC20 name.|
|`_symbol`|`string`|     ERC20 symbol.|


### mint

Allows the StandardBridge on this network to mint tokens.


```solidity
function mint(address _to, uint256 _amount)
    external
    virtual
    override(IOptimismMintableERC20, ILegacyMintableERC20)
    onlyBridge;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_to`|`address`|    Address to mint tokens to.|
|`_amount`|`uint256`|Amount of tokens to mint.|


### burn

Allows the StandardBridge on this network to burn tokens.


```solidity
function burn(address _from, uint256 _amount)
    external
    virtual
    override(IOptimismMintableERC20, ILegacyMintableERC20)
    onlyBridge;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|  Address to burn tokens from.|
|`_amount`|`uint256`|Amount of tokens to burn.|


### supportsInterface

ERC165 interface check function.


```solidity
function supportsInterface(bytes4 _interfaceId) external pure returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_interfaceId`|`bytes4`|Interface ID to check.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether or not the interface is supported by this contract.|


### l1Token

Legacy getter for the remote token. Use REMOTE_TOKEN going forward.


```solidity
function l1Token() public view returns (address);
```

### l2Bridge

Legacy getter for the bridge. Use BRIDGE going forward.


```solidity
function l2Bridge() public view returns (address);
```

### remoteToken

Legacy getter for REMOTE_TOKEN.


```solidity
function remoteToken() public view returns (address);
```

### bridge

Legacy getter for BRIDGE.


```solidity
function bridge() public view returns (address);
```

## Events
### Mint
Emitted whenever tokens are minted for an account.


```solidity
event Mint(address indexed account, uint256 amount);
```

### Burn
Emitted whenever tokens are burned from an account.


```solidity
event Burn(address indexed account, uint256 amount);
```

