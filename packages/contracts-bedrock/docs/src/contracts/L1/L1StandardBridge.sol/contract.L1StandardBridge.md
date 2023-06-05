# L1StandardBridge
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L1/L1StandardBridge.sol)

**Inherits:**
[StandardBridge](/contracts/universal/StandardBridge.sol/abstract.StandardBridge.md), [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The L1StandardBridge is responsible for transfering ETH and ERC20 tokens between L1 and
L2. In the case that an ERC20 token is native to L1, it will be escrowed within this
contract. If the ERC20 token is native to L2, it will be burnt. Before Bedrock, ETH was
stored within this contract. After Bedrock, ETH is instead stored inside the
OptimismPortal contract.
NOTE: this contract is not intended to support all variations of ERC20 tokens. Examples
of some token types that may not be properly supported by this contract include, but are
not limited to: tokens with transfer fees, rebasing tokens, and tokens with blocklists.


## Functions
### constructor


```solidity
constructor(address payable _messenger)
    Semver(1, 1, 0)
    StandardBridge(_messenger, payable(Predeploys.L2_STANDARD_BRIDGE));
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_messenger`|`address payable`|Address of the L1CrossDomainMessenger.|


### receive

Allows EOAs to bridge ETH by sending directly to the bridge.


```solidity
receive() external payable override onlyEOA;
```

### depositETH

Deposits some amount of ETH into the sender's account on L2.


```solidity
function depositETH(uint32 _minGasLimit, bytes calldata _extraData) external payable onlyEOA;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_minGasLimit`|`uint32`|Minimum gas limit for the deposit message on L2.|
|`_extraData`|`bytes`|  Optional data to forward to L2. Data supplied here will not be used to execute any code on L2 and is only emitted as extra data for the convenience of off-chain tooling.|


### depositETHTo

Deposits some amount of ETH into a target account on L2.
Note that if ETH is sent to a contract on L2 and the call fails, then that ETH will
be locked in the L2StandardBridge. ETH may be recoverable if the call can be
successfully replayed by increasing the amount of gas supplied to the call. If the
call will fail for any amount of gas, then the ETH will be locked permanently.


```solidity
function depositETHTo(address _to, uint32 _minGasLimit, bytes calldata _extraData) external payable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_to`|`address`|         Address of the recipient on L2.|
|`_minGasLimit`|`uint32`|Minimum gas limit for the deposit message on L2.|
|`_extraData`|`bytes`|  Optional data to forward to L2. Data supplied here will not be used to execute any code on L2 and is only emitted as extra data for the convenience of off-chain tooling.|


### depositERC20

Deposits some amount of ERC20 tokens into the sender's account on L2.


```solidity
function depositERC20(
    address _l1Token,
    address _l2Token,
    uint256 _amount,
    uint32 _minGasLimit,
    bytes calldata _extraData
) external virtual onlyEOA;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l1Token`|`address`|    Address of the L1 token being deposited.|
|`_l2Token`|`address`|    Address of the corresponding token on L2.|
|`_amount`|`uint256`|     Amount of the ERC20 to deposit.|
|`_minGasLimit`|`uint32`|Minimum gas limit for the deposit message on L2.|
|`_extraData`|`bytes`|  Optional data to forward to L2. Data supplied here will not be used to execute any code on L2 and is only emitted as extra data for the convenience of off-chain tooling.|


### depositERC20To

Deposits some amount of ERC20 tokens into a target account on L2.


```solidity
function depositERC20To(
    address _l1Token,
    address _l2Token,
    address _to,
    uint256 _amount,
    uint32 _minGasLimit,
    bytes calldata _extraData
) external virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l1Token`|`address`|    Address of the L1 token being deposited.|
|`_l2Token`|`address`|    Address of the corresponding token on L2.|
|`_to`|`address`|         Address of the recipient on L2.|
|`_amount`|`uint256`|     Amount of the ERC20 to deposit.|
|`_minGasLimit`|`uint32`|Minimum gas limit for the deposit message on L2.|
|`_extraData`|`bytes`|  Optional data to forward to L2. Data supplied here will not be used to execute any code on L2 and is only emitted as extra data for the convenience of off-chain tooling.|


### finalizeETHWithdrawal

Finalizes a withdrawal of ETH from L2.


```solidity
function finalizeETHWithdrawal(address _from, address _to, uint256 _amount, bytes calldata _extraData)
    external
    payable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|     Address of the withdrawer on L2.|
|`_to`|`address`|       Address of the recipient on L1.|
|`_amount`|`uint256`|   Amount of ETH to withdraw.|
|`_extraData`|`bytes`|Optional data forwarded from L2.|


### finalizeERC20Withdrawal

Finalizes a withdrawal of ERC20 tokens from L2.


```solidity
function finalizeERC20Withdrawal(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _extraData
) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l1Token`|`address`|  Address of the token on L1.|
|`_l2Token`|`address`|  Address of the corresponding token on L2.|
|`_from`|`address`|     Address of the withdrawer on L2.|
|`_to`|`address`|       Address of the recipient on L1.|
|`_amount`|`uint256`|   Amount of the ERC20 to withdraw.|
|`_extraData`|`bytes`|Optional data forwarded from L2.|


### l2TokenBridge

Retrieves the access of the corresponding L2 bridge contract.


```solidity
function l2TokenBridge() external view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the corresponding L2 bridge contract.|


### _initiateETHDeposit

Internal function for initiating an ETH deposit.


```solidity
function _initiateETHDeposit(address _from, address _to, uint32 _minGasLimit, bytes memory _extraData) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|       Address of the sender on L1.|
|`_to`|`address`|         Address of the recipient on L2.|
|`_minGasLimit`|`uint32`|Minimum gas limit for the deposit message on L2.|
|`_extraData`|`bytes`|  Optional data to forward to L2.|


### _initiateERC20Deposit

Internal function for initiating an ERC20 deposit.


```solidity
function _initiateERC20Deposit(
    address _l1Token,
    address _l2Token,
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
|`_l1Token`|`address`|    Address of the L1 token being deposited.|
|`_l2Token`|`address`|    Address of the corresponding token on L2.|
|`_from`|`address`|       Address of the sender on L1.|
|`_to`|`address`|         Address of the recipient on L2.|
|`_amount`|`uint256`|     Amount of the ERC20 to deposit.|
|`_minGasLimit`|`uint32`|Minimum gas limit for the deposit message on L2.|
|`_extraData`|`bytes`|  Optional data to forward to L2.|


### _emitETHBridgeInitiated

Emits the legacy ETHDepositInitiated event followed by the ETHBridgeInitiated event.
This is necessary for backwards compatibility with the legacy bridge.


```solidity
function _emitETHBridgeInitiated(address _from, address _to, uint256 _amount, bytes memory _extraData)
    internal
    override;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|     Address of the sender.|
|`_to`|`address`|       Address of the receiver.|
|`_amount`|`uint256`|   Amount of ETH sent.|
|`_extraData`|`bytes`|Extra data sent with the transaction.|


### _emitETHBridgeFinalized

Emits the legacy ETHWithdrawalFinalized event followed by the ETHBridgeFinalized
event. This is necessary for backwards compatibility with the legacy bridge.


```solidity
function _emitETHBridgeFinalized(address _from, address _to, uint256 _amount, bytes memory _extraData)
    internal
    override;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|     Address of the sender.|
|`_to`|`address`|       Address of the receiver.|
|`_amount`|`uint256`|   Amount of ETH sent.|
|`_extraData`|`bytes`|Extra data sent with the transaction.|


### _emitERC20BridgeInitiated

Emits the legacy ERC20DepositInitiated event followed by the ERC20BridgeInitiated
event. This is necessary for backwards compatibility with the legacy bridge.


```solidity
function _emitERC20BridgeInitiated(
    address _localToken,
    address _remoteToken,
    address _from,
    address _to,
    uint256 _amount,
    bytes memory _extraData
) internal override;
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

Emits the legacy ERC20WithdrawalFinalized event followed by the ERC20BridgeFinalized
event. This is necessary for backwards compatibility with the legacy bridge.


```solidity
function _emitERC20BridgeFinalized(
    address _localToken,
    address _remoteToken,
    address _from,
    address _to,
    uint256 _amount,
    bytes memory _extraData
) internal override;
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
### ETHDepositInitiated
Emitted whenever a deposit of ETH from L1 into L2 is initiated.


```solidity
event ETHDepositInitiated(address indexed from, address indexed to, uint256 amount, bytes extraData);
```

### ETHWithdrawalFinalized
Emitted whenever a withdrawal of ETH from L2 to L1 is finalized.


```solidity
event ETHWithdrawalFinalized(address indexed from, address indexed to, uint256 amount, bytes extraData);
```

### ERC20DepositInitiated
Emitted whenever an ERC20 deposit is initiated.


```solidity
event ERC20DepositInitiated(
    address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData
);
```

### ERC20WithdrawalFinalized
Emitted whenever an ERC20 withdrawal is finalized.


```solidity
event ERC20WithdrawalFinalized(
    address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData
);
```

