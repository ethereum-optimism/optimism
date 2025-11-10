// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";

/// @title SuperchainRevSharesCalculator
/// @notice Calculator for Superchain revenue share. It pays the greater amount between 2.5% of
///         gross revenue or 15% of net revenue (gross minus L1 fees) to the configured share recipient.
///         The second configured recipient receives the remaining revenue.
contract SuperchainRevSharesCalculator is ISemver, ISharesCalculator {
    /// @notice Emitted when the share recipient is updated.
    /// @param oldShareRecipient The old share recipient address.
    /// @param newShareRecipient The new share recipient address.
    event ShareRecipientUpdated(address indexed oldShareRecipient, address indexed newShareRecipient);

    /// @notice Emitted when the remainder recipient is updated.
    /// @param oldRemainderRecipient The old remainder recipient address.
    /// @param newRemainderRecipient The new remainder recipient address.
    event RemainderRecipientUpdated(address indexed oldRemainderRecipient, address indexed newRemainderRecipient);

    /// @notice Thrown when the caller is not the ProxyAdmin owner.
    error SharesCalculator_OnlyProxyAdminOwner();

    /// @notice Thrown when the gross share is zero.
    error SharesCalculator_ZeroGrossShare();

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Basis points scale.
    uint32 public constant BASIS_POINT_SCALE = 10_000;

    /// @notice Gross revenue share in basis points (2.5%).
    uint32 public constant GROSS_SHARE_BPS = 250;

    /// @notice Net revenue share in basis points (15%).
    uint32 public constant NET_SHARE_BPS = 1_500;

    /// @notice Address that receives the Superchain revenue share.
    address payable public shareRecipient;

    /// @notice Address that receives the remainder of the revenue.
    address payable public remainderRecipient;

    /// @notice Constructs the contract with an initial configuration.
    /// @param _shareRecipient Recipient of the Superchain revenue share.
    /// @param _remainderRecipient Recipient of the remainder.
    constructor(address payable _shareRecipient, address payable _remainderRecipient) {
        shareRecipient = _shareRecipient;
        remainderRecipient = _remainderRecipient;
    }

    /// @notice Returns the recipients and amounts for fee distribution.
    /// @dev The recipients returned MUST be able to receive ether or FeeSplitter#disburseFees will fail.
    /// @param _sequencerFeeRevenue Revenue from sequencer fees.
    /// @param _baseFeeRevenue Revenue from base fees.
    /// @param _operatorFeeRevenue Revenue from operator fees.
    /// @param _l1FeeRevenue Revenue from L1 fees.
    /// @return shareInfo_ Array of ShareInfo structs containing recipients and amounts.
    function getRecipientsAndAmounts(
        uint256 _sequencerFeeRevenue,
        uint256 _baseFeeRevenue,
        uint256 _operatorFeeRevenue,
        uint256 _l1FeeRevenue
    )
        external
        view
        returns (ShareInfo[] memory shareInfo_)
    {
        // Two recipients: share recipient first (explicit amount), remainder recipient second (0; FeeSplitter sends
        // remainder)
        shareInfo_ = new ShareInfo[](2);
        shareInfo_[0] = ShareInfo({ recipient: shareRecipient, amount: 0 });
        shareInfo_[1] = ShareInfo({ recipient: remainderRecipient, amount: 0 });

        // Gross component: 2.5% of total revenue.
        uint256 grossRevenue = _sequencerFeeRevenue + _baseFeeRevenue + _operatorFeeRevenue + _l1FeeRevenue;
        uint256 grossShare = (grossRevenue * uint256(GROSS_SHARE_BPS)) / BASIS_POINT_SCALE;

        // Ensure gross share is greater than zero
        if (grossShare == 0) {
            revert SharesCalculator_ZeroGrossShare();
        }

        // Net component: 15% of (total - L1 fees).
        uint256 netRevenue = grossRevenue - _l1FeeRevenue;
        uint256 netShare = (netRevenue * uint256(NET_SHARE_BPS)) / BASIS_POINT_SCALE;

        uint256 amountToShareRecipient = grossShare > netShare ? grossShare : netShare;

        // Set the share amount and the remainder.
        shareInfo_[0].amount = amountToShareRecipient;
        shareInfo_[1].amount = grossRevenue - amountToShareRecipient;
    }

    /// @notice Sets the share recipient. Only callable by the ProxyAdmin owner.
    /// @param _newShareRecipient The new share recipient address.
    function setShareRecipient(address payable _newShareRecipient) external {
        if (msg.sender != IProxyAdmin(Predeploys.PROXY_ADMIN).owner()) {
            revert SharesCalculator_OnlyProxyAdminOwner();
        }
        address oldShareRecipient = shareRecipient;
        shareRecipient = _newShareRecipient;
        emit ShareRecipientUpdated(oldShareRecipient, _newShareRecipient);
    }

    /// @notice Sets the remainder recipient. Only callable by the ProxyAdmin owner.
    /// @param _newRemainderRecipient The new remainder recipient address.
    function setRemainderRecipient(address payable _newRemainderRecipient) external {
        if (msg.sender != IProxyAdmin(Predeploys.PROXY_ADMIN).owner()) {
            revert SharesCalculator_OnlyProxyAdminOwner();
        }
        address oldRemainderRecipient = remainderRecipient;
        remainderRecipient = _newRemainderRecipient;
        emit RemainderRecipientUpdated(oldRemainderRecipient, _newRemainderRecipient);
    }
}
