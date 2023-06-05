# L2ERC721Bridge_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L2ERC721Bridge.t.sol)

**Inherits:**
[Messenger_Initializer](/contracts/test/CommonTest.t.sol/contract.Messenger_Initializer.md)


## State Variables
### localToken

```solidity
TestMintableERC721 internal localToken;
```


### remoteToken

```solidity
TestERC721 internal remoteToken;
```


### bridge

```solidity
L2ERC721Bridge internal bridge;
```


### otherBridge

```solidity
address internal constant otherBridge = address(0x3456);
```


### tokenId

```solidity
uint256 internal constant tokenId = 1;
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_constructor_succeeds


```solidity
function test_constructor_succeeds() public;
```

### test_bridgeERC721_succeeds


```solidity
function test_bridgeERC721_succeeds() public;
```

### test_bridgeERC721_fromContract_reverts


```solidity
function test_bridgeERC721_fromContract_reverts() external;
```

### test_bridgeERC721_localTokenZeroAddress_reverts


```solidity
function test_bridgeERC721_localTokenZeroAddress_reverts() external;
```

### test_bridgeERC721_remoteTokenZeroAddress_reverts


```solidity
function test_bridgeERC721_remoteTokenZeroAddress_reverts() external;
```

### test_bridgeERC721_wrongOwner_reverts


```solidity
function test_bridgeERC721_wrongOwner_reverts() external;
```

### test_bridgeERC721To_succeeds


```solidity
function test_bridgeERC721To_succeeds() external;
```

### test_bridgeERC721To_localTokenZeroAddress_reverts


```solidity
function test_bridgeERC721To_localTokenZeroAddress_reverts() external;
```

### test_bridgeERC721To_remoteTokenZeroAddress_reverts


```solidity
function test_bridgeERC721To_remoteTokenZeroAddress_reverts() external;
```

### test_bridgeERC721To_wrongOwner_reverts


```solidity
function test_bridgeERC721To_wrongOwner_reverts() external;
```

### test_finalizeBridgeERC721_succeeds


```solidity
function test_finalizeBridgeERC721_succeeds() external;
```

### test_finalizeBridgeERC721_interfaceNotCompliant_reverts


```solidity
function test_finalizeBridgeERC721_interfaceNotCompliant_reverts() external;
```

### test_finalizeBridgeERC721_notViaLocalMessenger_reverts


```solidity
function test_finalizeBridgeERC721_notViaLocalMessenger_reverts() external;
```

### test_finalizeBridgeERC721_notFromRemoteMessenger_reverts


```solidity
function test_finalizeBridgeERC721_notFromRemoteMessenger_reverts() external;
```

### test_finalizeBridgeERC721_selfToken_reverts


```solidity
function test_finalizeBridgeERC721_selfToken_reverts() external;
```

### test_finalizeBridgeERC721_alreadyExists_reverts


```solidity
function test_finalizeBridgeERC721_alreadyExists_reverts() external;
```

## Events
### ERC721BridgeInitiated

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

