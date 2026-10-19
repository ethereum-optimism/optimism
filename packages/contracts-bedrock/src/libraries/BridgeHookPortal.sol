// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";

// Interfaces
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";

/// @title BridgeHookPortal
/// @notice The OptimismPortal's side of the bridge hook. It builds each deposit item, asks the
///         hook and commits to the deposits the hook holds.
library BridgeHookPortal {
    /// @notice The Portal's bridge hook state.
    /// @custom:field hook             Address of the bridge hook, zero if none is set.
    /// @custom:field inHook           Whether the hook is being called.
    /// @custom:field depositNonce     Counter that makes each screened deposit unique.
    /// @custom:field outstandingItems Number of items the hook holds.
    /// @custom:field heldDeposits     Held deposits, keyed by item identifier.
    struct State {
        IBridgeHook hook;
        bool inHook;
        uint64 depositNonce;
        uint64 outstandingItems;
        mapping(bytes32 => bool) heldDeposits;
    }

    /// @notice Emitted when the bridge hook holds a deposit.
    /// @param id Identifier of the held deposit.
    event DepositHeld(bytes32 indexed id);

    /// @notice Thrown when the hook calls back into the Portal.
    error OptimismPortal_NoReentrancy();

    /// @notice Thrown when the caller is not the bridge hook.
    error OptimismPortal_NotBridgeHook();

    /// @notice Thrown when completing an item that is not held.
    error OptimismPortal_NotHeld();

    /// @notice Thrown when the ETH sent with a completion does not match the held item.
    error OptimismPortal_ValueMismatch();

    /// @notice Asks the hook about a deposit. On a hold, commits to its terms and hands its value
    ///         to the hook.
    /// @param self        The Portal's bridge hook state.
    /// @param _from       Sender of the deposit, aliased if it is a contract.
    /// @param _to         Target address on L2.
    /// @param _value      ETH value to send to the recipient.
    /// @param _gasLimit   Amount of L2 gas purchased.
    /// @param _isCreation Whether or not the transaction is a contract creation.
    /// @param _data       Data to trigger the recipient with.
    /// @return pass_ True if the deposit proceeds.
    function screenDeposit(
        State storage self,
        address _from,
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        internal
        returns (bool pass_)
    {
        if (self.inHook) revert OptimismPortal_NoReentrancy();

        BridgeHookTypes.Item memory item;
        item.direction = BridgeHookTypes.Direction.Deposit;
        item.asset = BridgeHookTypes.Asset.ETH;
        item.from = _from;
        item.aliased = _from != msg.sender;
        item.to = _to;
        item.ethAmount = msg.value;
        item.l2Value = _value;
        item.gasLimit = _gasLimit;
        item.isCreation = _isCreation;
        item.data = _data;
        item.nonce = self.depositNonce++;

        self.inHook = true;
        pass_ = self.hook.screenDeposit(item);
        if (!pass_) {
            bytes32 id = BridgeHookHashing.hashItem(item);
            self.heldDeposits[id] = true;
            self.outstandingItems++;
            emit DepositHeld(id);

            self.hook.holdDeposit{ value: msg.value }(item);
        }
        self.inHook = false;
    }

    /// @notice Checks a held deposit against its commitment and deletes the commitment. Only the
    ///         hook can complete. The Portal then emits the deposit.
    /// @param self  The Portal's bridge hook state.
    /// @param _item The held deposit.
    function completeDeposit(State storage self, BridgeHookTypes.Item memory _item) internal {
        if (msg.sender != address(self.hook)) revert OptimismPortal_NotBridgeHook();
        if (self.inHook) revert OptimismPortal_NoReentrancy();

        bytes32 id = BridgeHookHashing.hashItem(_item);
        if (!self.heldDeposits[id]) revert OptimismPortal_NotHeld();
        if (msg.value != _item.ethAmount) revert OptimismPortal_ValueMismatch();

        delete self.heldDeposits[id];
        self.outstandingItems--;
    }
}
