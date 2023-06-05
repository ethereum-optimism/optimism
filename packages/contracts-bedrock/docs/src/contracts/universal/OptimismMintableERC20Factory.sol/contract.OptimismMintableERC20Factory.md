# OptimismMintableERC20Factory
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/OptimismMintableERC20Factory.sol)

**Inherits:**
[Semver](/contracts/universal/Semver.sol/contract.Semver.md)

OptimismMintableERC20Factory is a factory contract that generates OptimismMintableERC20
contracts on the network it's deployed to. Simplifies the deployment process for users
who may be less familiar with deploying smart contracts. Designed to be backwards
compatible with the older StandardL2ERC20Factory contract.


## State Variables
### BRIDGE
Address of the StandardBridge on this chain.


```solidity
address public immutable BRIDGE;
```


## Functions
### constructor

The semver MUST be bumped any time that there is a change in
the OptimismMintableERC20 token contract since this contract
is responsible for deploying OptimismMintableERC20 contracts.


```solidity
constructor(address _bridge) Semver(1, 1, 0);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bridge`|`address`|Address of the StandardBridge on this chain.|


### createStandardL2Token

Creates an instance of the OptimismMintableERC20 contract. Legacy version of the
newer createOptimismMintableERC20 function, which has a more intuitive name.


```solidity
function createStandardL2Token(address _remoteToken, string memory _name, string memory _symbol)
    external
    returns (address);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_remoteToken`|`address`|Address of the token on the remote chain.|
|`_name`|`string`|       ERC20 name.|
|`_symbol`|`string`|     ERC20 symbol.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the newly created token.|


### createOptimismMintableERC20

Creates an instance of the OptimismMintableERC20 contract.


```solidity
function createOptimismMintableERC20(address _remoteToken, string memory _name, string memory _symbol)
    public
    returns (address);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_remoteToken`|`address`|Address of the token on the remote chain.|
|`_name`|`string`|       ERC20 name.|
|`_symbol`|`string`|     ERC20 symbol.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the newly created token.|


## Events
### StandardL2TokenCreated
Emitted whenever a new OptimismMintableERC20 is created. Legacy version of the newer
OptimismMintableERC20Created event. We recommend relying on that event instead.


```solidity
event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken);
```

### OptimismMintableERC20Created
Emitted whenever a new OptimismMintableERC20 is created.


```solidity
event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer);
```

