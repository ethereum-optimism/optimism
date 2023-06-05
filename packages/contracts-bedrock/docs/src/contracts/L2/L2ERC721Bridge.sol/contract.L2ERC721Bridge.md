# L2ERC721Bridge
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/L2ERC721Bridge.sol)

**Inherits:**
[ERC721Bridge](/contracts/universal/ERC721Bridge.sol/abstract.ERC721Bridge.md), [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The L2 ERC721 bridge is a contract which works together with the L1 ERC721 bridge to
make it possible to transfer ERC721 tokens from Ethereum to Optimism. This contract
acts as a minter for new tokens when it hears about deposits into the L1 ERC721 bridge.
This contract also acts as a burner for tokens being withdrawn.
WARNING**: Do not bridge an ERC721 that was originally deployed on Optimism. This
bridge ONLY supports ERC721s originally deployed on Ethereum. Users will need to
wait for the one-week challenge period to elapse before their Optimism-native NFT
can be refunded on L2.


## Functions
### constructor


```solidity
constructor(address _messenger, address _otherBridge) Semver(1, 1, 0) ERC721Bridge(_messenger, _otherBridge);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_messenger`|`address`|  Address of the CrossDomainMessenger on this network.|
|`_otherBridge`|`address`|Address of the ERC721 bridge on the other network.|


### finalizeBridgeERC721

Completes an ERC721 bridge from the other domain and sends the ERC721 token to the
recipient on this domain.


```solidity
function finalizeBridgeERC721(
    address _localToken,
    address _remoteToken,
    address _from,
    address _to,
    uint256 _tokenId,
    bytes calldata _extraData
) external onlyOtherBridge;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_localToken`|`address`| Address of the ERC721 token on this domain.|
|`_remoteToken`|`address`|Address of the ERC721 token on the other domain.|
|`_from`|`address`|       Address that triggered the bridge on the other domain.|
|`_to`|`address`|         Address to receive the token on this domain.|
|`_tokenId`|`uint256`|    ID of the token being deposited.|
|`_extraData`|`bytes`|  Optional data to forward to L1. Data supplied here will not be used to execute any code on L1 and is only emitted as extra data for the convenience of off-chain tooling.|


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
) internal override;
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


