// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;
    
/// @title ISharesCalculator
/// @notice Interface for a contract that calculates the recipients and amounts for fee distribution.
/// @dev Meant to be called by the FeeSplitter contract.
interface ISharesCalculator {
    /// @notice Struct to hold the recipient and amount for each fee share.
    /// @param recipient The address that will receive the fee share
    /// @param amount The amount of ETH to be sent to the recipient
    struct ShareInfo {
        address payable recipient;
        uint256 amount;
    }

    /// @notice Returns the recipients and amounts for fee distribution.
    /// @dev Any implementation MUST return ShareInfo where the sum of all amounts equals
    /// the total revenue (sum of all vault balances) as it will revert otherwise
    /// @param _sequencerFeeRevenue Balance of the sequencer fee vault.
    /// @param _baseFeeRevenue Balance of the base fee vault.
    /// @param _operatorFeeRevenue Balance of the operator fee vault.
    /// @param _l1FeeRevenue Balance of the L1 fee vault.
    /// @return shareInfo Array of ShareInfo structs containing recipients and amounts.
    function getRecipientsAndAmounts(
        uint256 _sequencerFeeRevenue,
        uint256 _baseFeeRevenue,
        uint256 _operatorFeeRevenue,
        uint256 _l1FeeRevenue
    )
        external
        view
        returns (ShareInfo[] memory shareInfo);
}