// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title BridgeHookTypes
/// @notice Contains the types shared by the OptimismPortal and its bridge hook.
library BridgeHookTypes {
    /// @notice Deposit from L1 to L2, or withdrawal from L2 to L1.
    enum Direction {
        Deposit,
        Withdrawal
    }

    /// @notice The asset an item moves. ERC20 is a token transfer between the standard bridges.
    enum Asset {
        ETH,
        ERC20
    }

    /// @notice A deposit or withdrawal with the sender, target, values and data recorded by the
    ///         Portal. Its fields determine its ID, and the full item is emitted when held and
    ///         supplied again for completion.
    /// @custom:field direction  Whether the item is a deposit or a withdrawal.
    /// @custom:field asset      ERC20 when the Portal recognizes a transfer between the standard
    ///                          bridges.
    /// @custom:field from       Deposit sender, aliased if it is a contract, or withdrawal sender.
    /// @custom:field aliased    Whether `from` is aliased.
    /// @custom:field to         Deposit or withdrawal target.
    /// @custom:field ethAmount  A deposit's msg.value or a withdrawal's value. Zero for a token
    ///                          transfer.
    /// @custom:field l2Value    ETH value of the L2 call. Deposits only.
    /// @custom:field gasLimit   Deposit or withdrawal gas limit.
    /// @custom:field isCreation Whether the deposit creates a contract.
    /// @custom:field data       Call or contract creation data. For token transfers, includes the
    ///                          token addresses and amount.
    /// @custom:field nonce      Deposit counter or withdrawal nonce.
    struct Item {
        Direction direction;
        Asset asset;
        address from;
        bool aliased;
        address to;
        uint256 ethAmount;
        uint256 l2Value;
        uint256 gasLimit;
        bool isCreation;
        bytes data;
        uint256 nonce;
    }
}
