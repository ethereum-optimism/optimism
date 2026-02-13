// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";

interface ISuperchainRevSharesCalculator is ISemver {
    event ShareRecipientUpdated(address indexed oldShareRecipient, address indexed newShareRecipient);
    event RemainderRecipientUpdated(address indexed oldRemainderRecipient, address indexed newRemainderRecipient);

    error SharesCalculator_OnlyProxyAdminOwner();
    error SharesCalculator_ZeroGrossShare();

    function BASIS_POINT_SCALE() external view returns (uint32);
    function GROSS_SHARE_BPS() external view returns (uint32);
    function NET_SHARE_BPS() external view returns (uint32);
    function shareRecipient() external view returns (address payable);
    function remainderRecipient() external view returns (address payable);

    function getRecipientsAndAmounts(
        uint256 _sequencerFeeRevenue,
        uint256 _baseFeeRevenue,
        uint256 _operatorFeeRevenue,
        uint256 _l1FeeRevenue
    )
        external
        view
        returns (ISharesCalculator.ShareInfo[] memory shareInfo_);

    function setShareRecipient(address payable _newShareRecipient) external;
    function setRemainderRecipient(address payable _newRemainderRecipient) external;

    function __constructor__(address payable _shareRecipient, address payable _remainderRecipient) external;
}
