// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";

interface ISuperchainRevSharesCalculator is ISemver {
    // Events
    event ShareRecipientUpdated(address indexed shareRecipient);
    event RemainderRecipientUpdated(address indexed remainderRecipient);

    // Errors
    error SharesCalculator_OnlyProxyAdminOwner();
    error SharesCalculator_ZeroGrossShare();

    // State variables
    function BASIS_POINT_SCALE() external view returns (uint32);
    function GROSS_SHARE_BPS() external view returns (uint32);
    function NET_SHARE_BPS() external view returns (uint32);
    function FEE_SPLITTER() external view returns (address);
    function shareRecipient() external view returns (address payable);
    function remainderRecipient() external view returns (address payable);

    // Functions
    function getRecipientsAndAmounts(
        uint256 _sequencerFeeRevenue,
        uint256 _baseFeeRevenue,
        uint256 _operatorFeeRevenue,
        uint256 _l1FeeRevenue
    )
        external
        view
        returns (ISharesCalculator.ShareInfo[] memory);

    function setShareRecipient(address payable _shareRecipient) external;
    function setRemainderRecipient(address payable _remainderRecipient) external;

    function __constructor__(address payable _shareRecipient, address payable _remainderRecipient) external;
}
