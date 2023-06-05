# LegacyCrossDomainUtils
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/LegacyCrossDomainUtils.sol)


## Functions
### encodeXDomainCalldata

Generates the correct cross domain calldata for a message.


```solidity
function encodeXDomainCalldata(address _target, address _sender, bytes memory _message, uint256 _messageNonce)
    internal
    pure
    returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_target`|`address`|Target contract address.|
|`_sender`|`address`|Message sender address.|
|`_message`|`bytes`|Message to send to the target.|
|`_messageNonce`|`uint256`|Nonce for the provided message.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|ABI encoded cross domain calldata.|


