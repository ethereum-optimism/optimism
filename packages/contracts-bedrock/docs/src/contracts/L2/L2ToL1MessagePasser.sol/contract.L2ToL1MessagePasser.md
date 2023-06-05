# L2ToL1MessagePasser
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/L2ToL1MessagePasser.sol)

**Inherits:**
[Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The L2ToL1MessagePasser is a dedicated contract where messages that are being sent from
L2 to L1 can be stored. The storage root of this contract is pulled up to the top level
of the L2 output to reduce the cost of proving the existence of sent messages.


## State Variables
### RECEIVE_DEFAULT_GAS_LIMIT
The L1 gas limit set when eth is withdrawn using the receive() function.


```solidity
uint256 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 100_000;
```


### MESSAGE_VERSION
Current message version identifier.


```solidity
uint16 public constant MESSAGE_VERSION = 1;
```


### sentMessages
Includes the message hashes for all withdrawals


```solidity
mapping(bytes32 => bool) public sentMessages;
```


### msgNonce
A unique value hashed with each withdrawal.


```solidity
uint240 internal msgNonce;
```


## Functions
### constructor


```solidity
constructor() Semver(1, 0, 0);
```

### receive

Allows users to withdraw ETH by sending directly to this contract.


```solidity
receive() external payable;
```

### burn

Removes all ETH held by this contract from the state. Used to prevent the amount of
ETH on L2 inflating when ETH is withdrawn. Currently only way to do this is to
create a contract and self-destruct it to itself. Anyone can call this function. Not
incentivized since this function is very cheap.


```solidity
function burn() external;
```

### initiateWithdrawal

Sends a message from L2 to L1.


```solidity
function initiateWithdrawal(address _target, uint256 _gasLimit, bytes memory _data) public payable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_target`|`address`|  Address to call on L1 execution.|
|`_gasLimit`|`uint256`|Minimum gas limit for executing the message on L1.|
|`_data`|`bytes`|    Data to forward to L1 target.|


### messageNonce

Retrieves the next message nonce. Message version will be added to the upper two
bytes of the message nonce. Message version allows us to treat messages as having
different structures.


```solidity
function messageNonce() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Nonce of the next message to be sent, with added message version.|


## Events
### MessagePassed
Emitted any time a withdrawal is initiated.


```solidity
event MessagePassed(
    uint256 indexed nonce,
    address indexed sender,
    address indexed target,
    uint256 value,
    uint256 gasLimit,
    bytes data,
    bytes32 withdrawalHash
);
```

### WithdrawerBalanceBurnt
Emitted when the balance of this contract is burned.


```solidity
event WithdrawerBalanceBurnt(uint256 indexed amount);
```

