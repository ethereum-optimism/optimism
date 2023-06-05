# StandardBridge
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/StandardBridge.sol)

StandardBridge is a base contract for the L1 and L2 standard ERC20 bridges. It handles
the core bridging logic, including escrowing tokens that are native to the local chain
and minting/burning tokens that are native to the remote chain.


## State Variables
### RECEIVE_DEFAULT_GAS_LIMIT
The L2 gas limit set when eth is depoisited using the receive() function.


```solidity
uint32 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 200_000;
```


### MESSENGER
Messenger contract on this domain.


```solidity
CrossDomainMessenger public immutable MESSENGER;
```


### OTHER_BRIDGE
Corresponding bridge on the other domain.


```solidity
StandardBridge public immutable OTHER_BRIDGE;
```


### spacer_0_0_20
Spacer for backwards compatibility.


```solidity
address private spacer_0_0_20;
```


### spacer_1_0_20
Spacer for backwards compatibility.


```solidity
address private spacer_1_0_20;
```


### deposits
Mapping that stores deposits for a given pair of local and remote tokens.


```solidity
mapping(address => mapping(address => uint256)) public deposits;
```


### __gap
Reserve extra slots (to a total of 50) in the storage layout for future upgrades.
A gap size of 47 was chosen here, so that the first slot used in a child contract
would be a multiple of 50.


```solidity
uint256[47] private __gap;
```


## Functions
### onlyEOA

Only allow EOAs to call the functions. Note that this is not safe against contracts
calling code within their constructors, but also doesn't really matter since we're
just trying to prevent users accidentally depositing with smart contract wallets.


```solidity
modifier onlyEOA();
```

### onlyOtherBridge

Ensures that the caller is a cross-chain message from the other bridge.


```solidity
modifier onlyOtherBridge();
```

### constructor


```solidity
constructor(address payable _messenger, address payable _otherBridge);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_messenger`|`address payable`|  Address of CrossDomainMessenger on this network.|
|`_otherBridge`|`address payable`|Address of the other StandardBridge contract.|


### receive

Allows EOAs to bridge ETH by sending directly to the bridge.
Must be implemented by contracts that inherit.


```solidity
receive() external payable virtual;
```

### messenger

Legacy getter for messenger contract.


```solidity
function messenger() external view returns (CrossDomainMessenger);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`CrossDomainMessenger`|Messenger contract on this domain.|


### bridgeETH

Sends ETH to the sender's address on the other chain.


```solidity
function bridgeETH(uint32 _minGasLimit, bytes calldata _extraData) public payable onlyEOA;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_minGasLimit`|`uint32`|Minimum amount of gas that the bridge can be relayed with.|
|`_extraData`|`bytes`|  Extra data to be sent with the transaction. Note that the recipient will not be triggered with this data, but it will be emitted and can be used to identify the transaction.|


### bridgeETHTo

Sends ETH to a receiver's address on the other chain. Note that if ETH is sent to a
smart contract and the call fails, the ETH will be temporarily locked in the
StandardBridge on the other chain until the call is replayed. If the call cannot be
replayed with any amount of gas (call always reverts), then the ETH will be
permanently locked in the StandardBridge on the other chain. ETH will also
be locked if the receiver is the other bridge, because finalizeBridgeETH will revert
in that case.


```solidity
function bridgeETHTo(address _to, uint32 _minGasLimit, bytes calldata _extraData) public payable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_to`|`address`|         Address of the receiver.|
|`_minGasLimit`|`uint32`|Minimum amount of gas that the bridge can be relayed with.|
|`_extraData`|`bytes`|  Extra data to be sent with the transaction. Note that the recipient will not be triggered with this data, but it will be emitted and can be used to identify the transaction.|


### bridgeERC20

Sends ERC20 tokens to the sender's address on the other chain. Note that if the
ERC20 token on the other chain does not recognize the local token as the correct
pair token, the ERC20 bridge will fail and the tokens will be returned to sender on
this chain.


```solidity
function bridgeERC20(
    address _localToken,
    address _remoteToken,
    uint256 _amount,
    uint32 _minGasLimit,
    bytes calldata _extraData
) public virtual onlyEOA;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC20 on this chain.|
|`_remoteToken`|`address`|Address of the corresponding token on the remote chain.|
|`_amount`|`uint256`|     Amount of local tokens to deposit.|
|`_minGasLimit`|`uint32`|Minimum amount of gas that the bridge can be relayed with.|
|`_extraData`|`bytes`|  Extra data to be sent with the transaction. Note that the recipient will not be triggered with this data, but it will be emitted and can be used to identify the transaction.|


### bridgeERC20To

Sends ERC20 tokens to a receiver's address on the other chain. Note that if the
ERC20 token on the other chain does not recognize the local token as the correct
pair token, the ERC20 bridge will fail and the tokens will be returned to sender on
this chain.


```solidity
function bridgeERC20To(
    address _localToken,
    address _remoteToken,
    address _to,
    uint256 _amount,
    uint32 _minGasLimit,
    bytes calldata _extraData
) public virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC20 on this chain.|
|`_remoteToken`|`address`|Address of the corresponding token on the remote chain.|
|`_to`|`address`|         Address of the receiver.|
|`_amount`|`uint256`|     Amount of local tokens to deposit.|
|`_minGasLimit`|`uint32`|Minimum amount of gas that the bridge can be relayed with.|
|`_extraData`|`bytes`|  Extra data to be sent with the transaction. Note that the recipient will not be triggered with this data, but it will be emitted and can be used to identify the transaction.|


### finalizeBridgeETH

Finalizes an ETH bridge on this chain. Can only be triggered by the other
StandardBridge contract on the remote chain.


```solidity
function finalizeBridgeETH(address _from, address _to, uint256 _amount, bytes calldata _extraData)
    public
    payable
    onlyOtherBridge;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|     Address of the sender.|
|`_to`|`address`|       Address of the receiver.|
|`_amount`|`uint256`|   Amount of ETH being bridged.|
|`_extraData`|`bytes`|Extra data to be sent with the transaction. Note that the recipient will not be triggered with this data, but it will be emitted and can be used to identify the transaction.|


### finalizeBridgeERC20

Finalizes an ERC20 bridge on this chain. Can only be triggered by the other
StandardBridge contract on the remote chain.


```solidity
function finalizeBridgeERC20(
    address _localToken,
    address _remoteToken,
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _extraData
) public onlyOtherBridge;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC20 on this chain.|
|`_remoteToken`|`address`|Address of the corresponding token on the remote chain.|
|`_from`|`address`|       Address of the sender.|
|`_to`|`address`|         Address of the receiver.|
|`_amount`|`uint256`|     Amount of the ERC20 being bridged.|
|`_extraData`|`bytes`|  Extra data to be sent with the transaction. Note that the recipient will not be triggered with this data, but it will be emitted and can be used to identify the transaction.|


### _initiateBridgeETH

Initiates a bridge of ETH through the CrossDomainMessenger.


```solidity
function _initiateBridgeETH(address _from, address _to, uint256 _amount, uint32 _minGasLimit, bytes memory _extraData)
    internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|       Address of the sender.|
|`_to`|`address`|         Address of the receiver.|
|`_amount`|`uint256`|     Amount of ETH being bridged.|
|`_minGasLimit`|`uint32`|Minimum amount of gas that the bridge can be relayed with.|
|`_extraData`|`bytes`|  Extra data to be sent with the transaction. Note that the recipient will not be triggered with this data, but it will be emitted and can be used to identify the transaction.|


### _initiateBridgeERC20

Sends ERC20 tokens to a receiver's address on the other chain.


```solidity
function _initiateBridgeERC20(
    address _localToken,
    address _remoteToken,
    address _from,
    address _to,
    uint256 _amount,
    uint32 _minGasLimit,
    bytes memory _extraData
) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC20 on this chain.|
|`_remoteToken`|`address`|Address of the corresponding token on the remote chain.|
|`_from`|`address`||
|`_to`|`address`|         Address of the receiver.|
|`_amount`|`uint256`|     Amount of local tokens to deposit.|
|`_minGasLimit`|`uint32`|Minimum amount of gas that the bridge can be relayed with.|
|`_extraData`|`bytes`|  Extra data to be sent with the transaction. Note that the recipient will not be triggered with this data, but it will be emitted and can be used to identify the transaction.|


### _isOptimismMintableERC20

Checks if a given address is an OptimismMintableERC20. Not perfect, but good enough.
Just the way we like it.


```solidity
function _isOptimismMintableERC20(address _token) internal view returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_token`|`address`|Address of the token to check.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|True if the token is an OptimismMintableERC20.|


### _isCorrectTokenPair

Checks if the "other token" is the correct pair token for the OptimismMintableERC20.
Calls can be saved in the future by combining this logic with
`_isOptimismMintableERC20`.


```solidity
function _isCorrectTokenPair(address _mintableToken, address _otherToken) internal view returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_mintableToken`|`address`|OptimismMintableERC20 to check against.|
|`_otherToken`|`address`|   Pair token to check.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|True if the other token is the correct pair token for the OptimismMintableERC20.|


### _emitETHBridgeInitiated

Emits the ETHBridgeInitiated event and if necessary the appropriate legacy event
when an ETH bridge is finalized on this chain.


```solidity
function _emitETHBridgeInitiated(address _from, address _to, uint256 _amount, bytes memory _extraData)
    internal
    virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|     Address of the sender.|
|`_to`|`address`|       Address of the receiver.|
|`_amount`|`uint256`|   Amount of ETH sent.|
|`_extraData`|`bytes`|Extra data sent with the transaction.|


### _emitETHBridgeFinalized

Emits the ETHBridgeFinalized and if necessary the appropriate legacy event when an
ETH bridge is finalized on this chain.


```solidity
function _emitETHBridgeFinalized(address _from, address _to, uint256 _amount, bytes memory _extraData)
    internal
    virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|     Address of the sender.|
|`_to`|`address`|       Address of the receiver.|
|`_amount`|`uint256`|   Amount of ETH sent.|
|`_extraData`|`bytes`|Extra data sent with the transaction.|


### _emitERC20BridgeInitiated

Emits the ERC20BridgeInitiated event and if necessary the appropriate legacy
event when an ERC20 bridge is initiated to the other chain.


```solidity
function _emitERC20BridgeInitiated(
    address _localToken,
    address _remoteToken,
    address _from,
    address _to,
    uint256 _amount,
    bytes memory _extraData
) internal virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC20 on this chain.|
|`_remoteToken`|`address`|Address of the ERC20 on the remote chain.|
|`_from`|`address`|       Address of the sender.|
|`_to`|`address`|         Address of the receiver.|
|`_amount`|`uint256`|     Amount of the ERC20 sent.|
|`_extraData`|`bytes`|  Extra data sent with the transaction.|


### _emitERC20BridgeFinalized

Emits the ERC20BridgeFinalized event and if necessary the appropriate legacy
event when an ERC20 bridge is initiated to the other chain.


```solidity
function _emitERC20BridgeFinalized(
    address _localToken,
    address _remoteToken,
    address _from,
    address _to,
    uint256 _amount,
    bytes memory _extraData
) internal virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC20 on this chain.|
|`_remoteToken`|`address`|Address of the ERC20 on the remote chain.|
|`_from`|`address`|       Address of the sender.|
|`_to`|`address`|         Address of the receiver.|
|`_amount`|`uint256`|     Amount of the ERC20 sent.|
|`_extraData`|`bytes`|  Extra data sent with the transaction.|


## Events
### ETHBridgeInitiated
Emitted when an ETH bridge is initiated to the other chain.


```solidity
event ETHBridgeInitiated(address indexed from, address indexed to, uint256 amount, bytes extraData);
```

### ETHBridgeFinalized
Emitted when an ETH bridge is finalized on this chain.


```solidity
event ETHBridgeFinalized(address indexed from, address indexed to, uint256 amount, bytes extraData);
```

### ERC20BridgeInitiated
Emitted when an ERC20 bridge is initiated to the other chain.


```solidity
event ERC20BridgeInitiated(
    address indexed localToken,
    address indexed remoteToken,
    address indexed from,
    address to,
    uint256 amount,
    bytes extraData
);
```

### ERC20BridgeFinalized
Emitted when an ERC20 bridge is finalized on this chain.


```solidity
event ERC20BridgeFinalized(
    address indexed localToken,
    address indexed remoteToken,
    address indexed from,
    address to,
    uint256 amount,
    bytes extraData
);
```

