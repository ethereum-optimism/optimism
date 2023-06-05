# Bridge_Initializer
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CommonTest.t.sol)

**Inherits:**
[Messenger_Initializer](/contracts/test/CommonTest.t.sol/contract.Messenger_Initializer.md)


## State Variables
### L1Bridge

```solidity
L1StandardBridge L1Bridge;
```


### L2Bridge

```solidity
L2StandardBridge L2Bridge;
```


### L2TokenFactory

```solidity
OptimismMintableERC20Factory L2TokenFactory;
```


### L1TokenFactory

```solidity
OptimismMintableERC20Factory L1TokenFactory;
```


### L1Token

```solidity
ERC20 L1Token;
```


### BadL1Token

```solidity
ERC20 BadL1Token;
```


### L2Token

```solidity
OptimismMintableERC20 L2Token;
```


### LegacyL2Token

```solidity
LegacyMintableERC20 LegacyL2Token;
```


### NativeL2Token

```solidity
ERC20 NativeL2Token;
```


### BadL2Token

```solidity
ERC20 BadL2Token;
```


### RemoteL1Token

```solidity
OptimismMintableERC20 RemoteL1Token;
```


## Functions
### setUp


```solidity
function setUp() public virtual override;
```

## Events
### ETHDepositInitiated

```solidity
event ETHDepositInitiated(address indexed from, address indexed to, uint256 amount, bytes data);
```

### ETHWithdrawalFinalized

```solidity
event ETHWithdrawalFinalized(address indexed from, address indexed to, uint256 amount, bytes data);
```

### ERC20DepositInitiated

```solidity
event ERC20DepositInitiated(
    address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data
);
```

### ERC20WithdrawalFinalized

```solidity
event ERC20WithdrawalFinalized(
    address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data
);
```

### WithdrawalInitiated

```solidity
event WithdrawalInitiated(
    address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data
);
```

### DepositFinalized

```solidity
event DepositFinalized(
    address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data
);
```

### DepositFailed

```solidity
event DepositFailed(
    address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes data
);
```

### ETHBridgeInitiated

```solidity
event ETHBridgeInitiated(address indexed from, address indexed to, uint256 amount, bytes data);
```

### ETHBridgeFinalized

```solidity
event ETHBridgeFinalized(address indexed from, address indexed to, uint256 amount, bytes data);
```

### ERC20BridgeInitiated

```solidity
event ERC20BridgeInitiated(
    address indexed localToken,
    address indexed remoteToken,
    address indexed from,
    address to,
    uint256 amount,
    bytes data
);
```

### ERC20BridgeFinalized

```solidity
event ERC20BridgeFinalized(
    address indexed localToken,
    address indexed remoteToken,
    address indexed from,
    address to,
    uint256 amount,
    bytes data
);
```

