# PortalSender
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/deployment/PortalSender.sol)

The PortalSender is a simple intermediate contract that will transfer the balance of the
L1StandardBridge to the OptimismPortal during the Bedrock migration.


## State Variables
### PORTAL
Address of the OptimismPortal contract.


```solidity
OptimismPortal public immutable PORTAL;
```


## Functions
### constructor


```solidity
constructor(OptimismPortal _portal);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_portal`|`OptimismPortal`|Address of the OptimismPortal contract.|


### donate

Sends balance of this contract to the OptimismPortal.


```solidity
function donate() public;
```

