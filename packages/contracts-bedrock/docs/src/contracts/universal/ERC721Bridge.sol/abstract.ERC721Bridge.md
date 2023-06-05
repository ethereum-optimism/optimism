# ERC721Bridge
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/ERC721Bridge.sol)

ERC721Bridge is a base contract for the L1 and L2 ERC721 bridges.


## State Variables
### MESSENGER
Messenger contract on this domain.


```solidity
CrossDomainMessenger public immutable MESSENGER;
```


### OTHER_BRIDGE
Address of the bridge on the other network.


```solidity
address public immutable OTHER_BRIDGE;
```


### __gap
Reserve extra slots (to a total of 50) in the storage layout for future upgrades.


```solidity
uint256[49] private __gap;
```


## Functions
### onlyOtherBridge

Ensures that the caller is a cross-chain message from the other bridge.


```solidity
modifier onlyOtherBridge();
```

### constructor


```solidity
constructor(address _messenger, address _otherBridge);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_messenger`|`address`|  Address of the CrossDomainMessenger on this network.|
|`_otherBridge`|`address`|Address of the ERC721 bridge on the other network.|


### messenger

Legacy getter for messenger contract.


```solidity
function messenger() external view returns (CrossDomainMessenger);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`CrossDomainMessenger`|Messenger contract on this domain.|


### otherBridge

Legacy getter for other bridge address.


```solidity
function otherBridge() external view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the bridge on the other network.|


### bridgeERC721

Initiates a bridge of an NFT to the caller's account on the other chain. Note that
this function can only be called by EOAs. Smart contract wallets should use the
`bridgeERC721To` function after ensuring that the recipient address on the remote
chain exists. Also note that the current owner of the token on this chain must
approve this contract to operate the NFT before it can be bridged.
WARNING**: Do not bridge an ERC721 that was originally deployed on Optimism. This
bridge only supports ERC721s originally deployed on Ethereum. Users will need to
wait for the one-week challenge period to elapse before their Optimism-native NFT
can be refunded on L2.


```solidity
function bridgeERC721(
    address _localToken,
    address _remoteToken,
    uint256 _tokenId,
    uint32 _minGasLimit,
    bytes calldata _extraData
) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC721 on this domain.|
|`_remoteToken`|`address`|Address of the ERC721 on the remote domain.|
|`_tokenId`|`uint256`|    Token ID to bridge.|
|`_minGasLimit`|`uint32`|Minimum gas limit for the bridge message on the other domain.|
|`_extraData`|`bytes`|  Optional data to forward to the other chain. Data supplied here will not be used to execute any code on the other chain and is only emitted as extra data for the convenience of off-chain tooling.|


### bridgeERC721To

Initiates a bridge of an NFT to some recipient's account on the other chain. Note
that the current owner of the token on this chain must approve this contract to
operate the NFT before it can be bridged.
WARNING**: Do not bridge an ERC721 that was originally deployed on Optimism. This
bridge only supports ERC721s originally deployed on Ethereum. Users will need to
wait for the one-week challenge period to elapse before their Optimism-native NFT
can be refunded on L2.


```solidity
function bridgeERC721To(
    address _localToken,
    address _remoteToken,
    address _to,
    uint256 _tokenId,
    uint32 _minGasLimit,
    bytes calldata _extraData
) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC721 on this domain.|
|`_remoteToken`|`address`|Address of the ERC721 on the remote domain.|
|`_to`|`address`|         Address to receive the token on the other domain.|
|`_tokenId`|`uint256`|    Token ID to bridge.|
|`_minGasLimit`|`uint32`|Minimum gas limit for the bridge message on the other domain.|
|`_extraData`|`bytes`|  Optional data to forward to the other chain. Data supplied here will not be used to execute any code on the other chain and is only emitted as extra data for the convenience of off-chain tooling.|


### _initiateBridgeERC721

Internal function for initiating a token bridge to the other domain.


```solidity
function _initiateBridgeERC721(
    address _localToken,
    address _remoteToken,
    address _from,
    address _to,
    uint256 _tokenId,
    uint32 _minGasLimit,
    bytes calldata _extraData
) internal virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC721 on this domain.|
|`_remoteToken`|`address`|Address of the ERC721 on the remote domain.|
|`_from`|`address`|       Address of the sender on this domain.|
|`_to`|`address`|         Address to receive the token on the other domain.|
|`_tokenId`|`uint256`|    Token ID to bridge.|
|`_minGasLimit`|`uint32`|Minimum gas limit for the bridge message on the other domain.|
|`_extraData`|`bytes`|  Optional data to forward to the other domain. Data supplied here will not be used to execute any code on the other domain and is only emitted as extra data for the convenience of off-chain tooling.|


## Events
### ERC721BridgeInitiated
Emitted when an ERC721 bridge to the other network is initiated.


```solidity
event ERC721BridgeInitiated(
    address indexed localToken,
    address indexed remoteToken,
    address indexed from,
    address to,
    uint256 tokenId,
    bytes extraData
);
```

### ERC721BridgeFinalized
Emitted when an ERC721 bridge from the other network is finalized.


```solidity
event ERC721BridgeFinalized(
    address indexed localToken,
    address indexed remoteToken,
    address indexed from,
    address to,
    uint256 tokenId,
    bytes extraData
);
```

