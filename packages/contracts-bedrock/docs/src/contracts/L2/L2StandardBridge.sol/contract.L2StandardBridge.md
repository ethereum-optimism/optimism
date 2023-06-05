# L2StandardBridge
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/L2StandardBridge.sol)

**Inherits:**
[StandardBridge](/contracts/universal/StandardBridge.sol/abstract.StandardBridge.md), [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The L2StandardBridge is responsible for transfering ETH and ERC20 tokens between L1 and
L2. In the case that an ERC20 token is native to L2, it will be escrowed within this
contract. If the ERC20 token is native to L1, it will be burnt.
NOTE: this contract is not intended to support all variations of ERC20 tokens. Examples
of some token types that may not be properly supported by this contract include, but are
not limited to: tokens with transfer fees, rebasing tokens, and tokens with blocklists.


## Functions
### constructor


```solidity
constructor(address payable _otherBridge)
    Semver(1, 1, 0)
    StandardBridge(payable(Predeploys.L2_CROSS_DOMAIN_MESSENGER), _otherBridge);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_otherBridge`|`address payable`|Address of the L1StandardBridge.|


### receive

Allows EOAs to bridge ETH by sending directly to the bridge.


```solidity
receive() external payable override onlyEOA;
```

### withdraw

Initiates a withdrawal from L2 to L1.
This function only works with OptimismMintableERC20 tokens or ether. Use the
`bridgeERC20` function to bridge native L2 tokens to L1.


```solidity
function withdraw(address _l2Token, uint256 _amount, uint32 _minGasLimit, bytes calldata _extraData)
    external
    payable
    virtual
    onlyEOA;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2Token`|`address`|    Address of the L2 token to withdraw.|
|`_amount`|`uint256`|     Amount of the L2 token to withdraw.|
|`_minGasLimit`|`uint32`|Minimum gas limit to use for the transaction.|
|`_extraData`|`bytes`|  Extra data attached to the withdrawal.|


### withdrawTo

Initiates a withdrawal from L2 to L1 to a target account on L1.
Note that if ETH is sent to a contract on L1 and the call fails, then that ETH will
be locked in the L1StandardBridge. ETH may be recoverable if the call can be
successfully replayed by increasing the amount of gas supplied to the call. If the
call will fail for any amount of gas, then the ETH will be locked permanently.
This function only works with OptimismMintableERC20 tokens or ether. Use the
`bridgeERC20To` function to bridge native L2 tokens to L1.


```solidity
function withdrawTo(address _l2Token, address _to, uint256 _amount, uint32 _minGasLimit, bytes calldata _extraData)
    external
    payable
    virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2Token`|`address`|    Address of the L2 token to withdraw.|
|`_to`|`address`|         Recipient account on L1.|
|`_amount`|`uint256`|     Amount of the L2 token to withdraw.|
|`_minGasLimit`|`uint32`|Minimum gas limit to use for the transaction.|
|`_extraData`|`bytes`|  Extra data attached to the withdrawal.|


### finalizeDeposit

Finalizes a deposit from L1 to L2. To finalize a deposit of ether, use address(0)
and the l1Token and the Legacy ERC20 ether predeploy address as the l2Token.


```solidity
function finalizeDeposit(
    address _l1Token,
    address _l2Token,
    address _from,
    address _to,
    uint256 _amount,
    bytes calldata _extraData
) external payable virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l1Token`|`address`|  Address of the L1 token to deposit.|
|`_l2Token`|`address`|  Address of the corresponding L2 token.|
|`_from`|`address`|     Address of the depositor.|
|`_to`|`address`|       Address of the recipient.|
|`_amount`|`uint256`|   Amount of the tokens being deposited.|
|`_extraData`|`bytes`|Extra data attached to the deposit.|


### l1TokenBridge

Retrieves the access of the corresponding L1 bridge contract.


```solidity
function l1TokenBridge() external view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the corresponding L1 bridge contract.|


### _initiateWithdrawal

Internal function to a withdrawal from L2 to L1 to a target account on L1.


```solidity
function _initiateWithdrawal(
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
|`_l2Token`|`address`|    Address of the L2 token to withdraw.|
|`_from`|`address`|       Address of the withdrawer.|
|`_to`|`address`|         Recipient account on L1.|
|`_amount`|`uint256`|     Amount of the L2 token to withdraw.|
|`_minGasLimit`|`uint32`|Minimum gas limit to use for the transaction.|
|`_extraData`|`bytes`|  Extra data attached to the withdrawal.|


### _emitETHBridgeInitiated

Emits the legacy WithdrawalInitiated event followed by the ETHBridgeInitiated event.
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

Emits the legacy DepositFinalized event followed by the ETHBridgeFinalized event.
This is necessary for backwards compatibility with the legacy bridge.


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

Emits the legacy WithdrawalInitiated event followed by the ERC20BridgeInitiated
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

Emits the legacy DepositFinalized event followed by the ERC20BridgeFinalized event.
This is necessary for backwards compatibility with the legacy bridge.


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
### WithdrawalInitiated
Emitted whenever a withdrawal from L2 to L1 is initiated.


```solidity
event WithdrawalInitiated(
    address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData
);
```

### DepositFinalized
Emitted whenever an ERC20 deposit is finalized.


```solidity
event DepositFinalized(
    address indexed l1Token, address indexed l2Token, address indexed from, address to, uint256 amount, bytes extraData
);
```

