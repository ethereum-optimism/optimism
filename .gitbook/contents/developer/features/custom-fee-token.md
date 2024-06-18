# Custom Fee Token

## Overview

Custom gas token allows for an L1-native ERC20 token to collateralize and act as the gas token on L2. This implementation is based on [Optimism's Custom Fee Token](https://specs.optimism.io/protocol/granite/custom-gas-token.html) and enhances its functionality by enabling the bridging of the L1 native token to L2 (ETH) as an ERC20 token.

**The codebase is located in the [custom-fee-token](https://github.com/bobanetwork/boba/tree/custom-fee-token) branch.** **The smart contract is UNDER audit. Please use it with caution.**

## Native Gas Tokens

By default, L2 OP Stack chains allow users to deposit ETH from L1 into the L2 chain, where it becomes the native L2 token used to pay for gas fees. However, chain operators wanted the ability to configure the stack to use a custom token for gas payments instead of ETH.

With custom gas tokens, chain operators can now specify an L1 ERC-20 token address when deploying their L2 chain contracts. When this L1 ERC-20 token is deposited, it becomes the native gas token on the L2, usable for gas fees.

### ETH Support

Our codebase allows you to configure ETH as an ERC-20 token on L2, enabling users to bridge ETH between L1 and L2. This feature is not supported by the standard OP Stack.

### Considerations

The custom gas paying token on L1 must adhere to the following constraints

- must be a valid ERC-20 token

* the number of decimals on the token MUST be exactly 18
* the name of the token MUST be less than or equal to 32 bytes
* symbol MUST be less than or equal to 32 bytes.

The ETH ERC20 token on L2 must adhere to the following constraints:

* must be a valid [OptimismMintableERC20](https://github.com/bobanetwork/boba/blob/custom-fee-token/packages/contracts-bedrock/src/legacy/LegacyMintableERC20.sol) token on L2
* the `l1Token` address MUST be `0x0000000000000000000000000000000000000000`

## Configuring the Gas Paying Token and L2 ETH Token

The gas-paying token and the L2 ETH token are set within the L1 `SystemConfig` smart contract. The gas-paying token is set during initialization and cannot be modified by the `SystemConfig` bytecode. The L2 ETH token is set during initialization and can be updated via the `setL2ETHToken` function if the L2 ETH token address is `address(0)`. Since the `SystemConfig` is proxied, it is always possible to modify the storage slot that holds the gas-paying token address and the L2 ETH token address directly during an upgrade.

If the address in the `GAS_PAYING_TOKEN_SLOT` slot for `SystemConfig` is `address(0)`, the system is configured to use `ether` as the gas paying token, and the getter for the token returns `ETHER_TOKEN_ADDRESS (0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE)` If the address in the `GAS_PAYING_TOKEN_SLOT` slot for `SystemConfig` is not `address(0)`, the system is configured to use a custom gas paying token, and the getter returns the address in the slot.

If the gas paying token is configured and the address in the `L2_ETH_TOKEN_SLOT` slot for `SymstemConfig` is `address(0)`, the system is configured to block the ETH deposit and withdrawal. If the address in the `L2_ETH_TOKEN_SLOT` slot for `SystemConfig` is not `address(0)`, the system is configured to allow the ETH deposit and withdrawal.

<figure><img src="../../../assets/feature gas paying token.png" alt=""><figcaption></figcaption></figure>

## Contract Modifications

### OptimismPortal

The `OptimismPortal` is updated with a new interface specifically for depositing custom tokens.

#### depositERC20Transaction

The `depositERC20Transaction` function is useful for sending custom gas tokens to L2. It is broken out into its own interface to maintain backwards compatibility with chains that use `ether`, to help simplify the implementation and make it explicit for callers that are trying to deposit an ERC20 native asset.

```solidity
function depositERC20Transaction(
    address _to,
    uint256 _mint,
    uint256 _value,
    uint64 _gasLimit,
    bool _isCreation,
    bytes memory _data
) public;
```

This function MUST revert when `ether` is the L2's native asset. It MUST not be `payable`, meaning that it will revert when `ether` is sent with a `CALL` to it. It uses a `transferFrom` flow, so users MUST first `approve()` the `OptimismPortal` before they can deposit tokens.

##### Function Arguments

The following table describes the arguments to `depositERC20Transaction`

| Name          | Type      | Description                                                  |
| ------------- | --------- | ------------------------------------------------------------ |
| `_to`         | `address` | The target of the deposit transaction                        |
| `_mint`       | `uint256` | The amount of token to deposit                               |
| `_value`      | `uint256` | The value of the deposit transaction, used to transfer native asset that is already on L2 from L1 |
| `_gasLimit`   | `uint64`  | The gas limit of the deposit transaction                     |
| `_isCreation` | `bool`    | Signifies the `_data` should be used with `CREATE`           |
| `_data`       | `bytes`   | The calldata of the deposit transaction                      |

#### depositTransaction

The `depositTransaction` function has been modified to support `ether` deposits when the custom fee token is enabled. If the custom fee token is enabled and `msg.value` is not zero, this function can only be called by an EOA (Externally Owned Account) to prevent incorrect minting on L2.

#### setGasPayingToken

This function MUST only be callable by the `SystemConfig`. When called, it creates a special deposit transaction from the `DEPOSITOR_ACCOUNT` that calls the `L1Block.setGasPayingToken` function. The ERC20 `name` and `symbol` are passed as `bytes32` to prevent the usage of dynamically sized `string`arguments.

```solidity
function setGasPayingToken(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) external;
```

#### SetL2ETHToken

This function MUST only be callable by the `SystemConfig`. When called, it creates a special deposit transaction from the `DEPOSITOR_ACCOUNT` that calls the `L1Block.setL2ETHToken` function. 

```solidity
function setL2ETHToken(address _token) external;
```

### L1CrossDomainMessenger

The `L1CrossDomainMessenger` contract exposes the `sendMintETHERC20Message` function, which can only be called by the `OptimismPortal` contract within the `depositTransaction` function. The `sendMintETHERC20Message` function constructs the payload for the `L2StandardBridge` to mint the ERC20 token.

### SystemConfig

The `SystemConfig` is the source of truth for the address of the custom gas token. It does on chain validation, stores information about the token and well as passes the information to L2.

#### initialize

The `SystemConfig` is modified to allow the addresses of the custom gas paying token and L2 ETH address to be set during the call to `initialize`.

#### setL2ETHToken

The `setL2ETHToken` function can be called only when the L2 ETH address has not been set and the custom fee token is enabled.

### L2ToL1MessagePasser

The `L2ToL1MessagePasser` has a new function called `initiateETHERC20Withdrawal` to initiate the `ether` withdrawal.

#### initiateETHERC20Withdrawal

The `initiateETHERC20Withdrawal` function can only be called by the `L2StandardBridge`. The `L2StandardBridge` initiates the `ether` withdrawal by first burning the `ether` ERC20 token. It then calls `initiateETHERC20Withdrawal` to emit the appropriate events, allowing the `OptimismPortal` contract to release the token.

## User Flow

The user flow for custom gas token chains is slightly different than for chains that use `ether` to pay for gas. The following tables highlight the methods that can be used for depositing and withdrawing the native asset. Not every interface is included.

#### When ETH is the Native Asset

| Scenario                                         | Method                                                       | Prerequisites                       |
| ------------------------------------------------ | ------------------------------------------------------------ | ----------------------------------- |
| Native Asset Send to Other Domain                | `L1StandardBridge.bridgeETH(uint32,bytes) payable`           | None                                |
| Native Asset and/or Message Send to Other Domain | `L1CrossDomainMessenger.sendMessage(address,bytes,uint32) payable` | None                                |
| Native Asset Deposit                             | `OptimismPortal.depositTransaction(address,uint256,uint64,bool,bytes) payable` | None                                |
| ERC20 Send to Other Domain                       | `L1StandardBridge.bridgeERC20(address,address,uint256,uint32,bytes)` | Approve `L1StandardBridge`for ERC20 |
| Native Asset Withdrawal                          | `L2ToL1MessagePasser.initiateWithdrawal(address,uint256,bytes) payable` |                                     |

There are multiple APIs for users to deposit or withdraw `ether`. Depending on the usecase, different APIs should be preferred. For a simple send of just `ether` with no calldata, the `OptimismPortal` or `L2ToL1MessagePasser` should be used directly. If sending with calldata and replayability on failed calls is desired, the `CrossDomainMessenger` should be used. Using the `StandardBridge` is the most expensive and has no real benefit for end users.

#### When an ERC20 Token is the Native Asset

| Scenario                   | Method                                                       | Prerequisites                           |
| -------------------------- | ------------------------------------------------------------ | --------------------------------------- |
| Native Asset Deposit       | `OptimismPortal.depositERC20Transaction(address,uint256,uint256,uint64,bool,bytes)` | Approve `OptimismPortal`for ERC20       |
| ERC20 Send to Other Domain | `L1StandardBridge.bridgeERC20(address,address,uint256,uint32,bytes)` | Approve `L1StandardBridge`for ERC20     |
| Native Asset Withdrawal    | `L2ToL1MessagePasser.initiateWithdrawal(address,uint256,bytes) payable` | None                                    |
| ETH Deposit                | `OptimismPortal.depositTransaction(address,uint256,uint64,bool,bytes) payable` | None                                    |
| ETH Withdrawal             | `L2StandardBridge.withdraw(address,uint256,uint32,bytes)`    | Approve `L2StandardBrige` for ETH ERC20 |

Users should deposit native asset by calling `depositERC20Transaction` on the `OptimismPortal`contract. Users must first `approve` the address of the `OptimismPortal` so that the `OptimismPortal`can use `transferFrom` to take ownership of the ERC20 asset. Users should withdraw value by calling the `L2ToL1MessagePasser` directly.

Users should deposit ETH by calling `depositTransaction` on the `OptimismPortal` contract or sending ETH this contract. Users should withdraw value by calling the `withdraw`  on the `L2StandardBridge` contract. Users must first `approve` the address of the `L2StandardBridge` so that the L2StandardBridge use `transferFrom` to take ownership of the ERC20 asset.

The following diagram shows the control flow for when a user sends `ether`.

<figure><img src="../../../assets/feature gas paying token eth.png" alt=""><figcaption></figcaption></figure>

## Technical Review

### Deposit

The deposit of the custom paying token can only be triggered in the `OptimismPortal` contract. The `depositERC20Transaction` function overrides the `_mint` value, allowing layer 2 to mint the native token.

The deposit of ETH can only be triggered in the `OptimismPortal` when the `L2ETHToken` is set. Users can either transfer ETH to `OptimismPortal` or call the `depositTransaction` function. The `depositTransaction` function retrieves the `opaqueData` in the `L1CrossDomainMessenger` contract. This `opaqueData` triggers the `RelayMessage` in the `L2CrossDomainMessenger` contract, allowing the `L2StandardBridge` to mint the L2 ETH ERC20 token.

### Withdrawal

The withdrawal of the custom paying token can only be triggered in the `L2ToL1MessagePasser` contract by either sending the custom paying token to this contract or calling the `initiateWithdrawal` function. The `_data` passed in the `MessagePassed` event is decoded in the L1 `OptimismPortal` contract. The `l2sender` decoded from the `_data` is `OptimismPortal`, allowing the `OptimismPortal` contract to transfer the custom paying token to the receiver's address.

The withdrawal of ETH can be triggered in the `L2StandardBridge` contract. The `L2StandardBridge` contract burns the token and calls the `L2ToL1MessagePasser` contract to forward the message to the L1 `OptimismPortal` contract. In this case, the `l2sender` in `_data` is `L2StandardBridge`, enabling the `OptimismPortal` contract to transfer ETH to the receiver's address.