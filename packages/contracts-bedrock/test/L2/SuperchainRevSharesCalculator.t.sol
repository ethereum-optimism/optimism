// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Interfaces
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";
import { ISuperchainRevSharesCalculator } from "interfaces/L2/ISuperchainRevSharesCalculator.sol";

// Libraries

/// @notice Base setup contract for SuperchainRevSharesCalculator tests.
contract SuperchainRevSharesCalculator_TestInit is CommonTest {
    uint256 internal constant BASIS_POINT_SCALE = 10_000;
    uint256 internal constant GROSS_SHARE_BPS = 250;
    uint256 internal constant NET_SHARE_BPS = 1_500;

    address payable shareRecipient;
    address payable remainderRecipient;

    event ShareRecipientUpdated(address indexed oldShareRecipient, address indexed newShareRecipient);
    event RemainderRecipientUpdated(address indexed oldRemainderRecipient, address indexed newRemainderRecipient);

    function setUp() public virtual override {
        // Enable revenue sharing before calling parent setUp
        super.enableRevenueShare();
        super.setUp();

        shareRecipient = payable(address(l1Withdrawer));
        remainderRecipient = payable(deploy.cfg().chainFeesRecipient());
    }
}

/// @notice Tests for SuperchainRevSharesCalculator constructor.
contract SuperchainRevSharesCalculator_Constructor_Test is SuperchainRevSharesCalculator_TestInit {
    /// @notice Tests that constructor
    function test_constructor_succeeds() external view {
        // Verify constants are set correctly on the deployed calculator
        assertEq(superchainRevSharesCalculator.version(), "1.0.0");
        assertEq(superchainRevSharesCalculator.BASIS_POINT_SCALE(), BASIS_POINT_SCALE);
        assertEq(superchainRevSharesCalculator.GROSS_SHARE_BPS(), GROSS_SHARE_BPS);
        assertEq(superchainRevSharesCalculator.NET_SHARE_BPS(), NET_SHARE_BPS);

        // Verify share and remainder recipients are set
        assertEq(address(superchainRevSharesCalculator.shareRecipient()), address(shareRecipient));
        assertEq(address(superchainRevSharesCalculator.remainderRecipient()), address(remainderRecipient));
    }
}

/// @notice Tests for SuperchainRevSharesCalculator setShareRecipient function success cases.
contract SuperchainRevSharesCalculator_SetShareRecipient_Test is SuperchainRevSharesCalculator_TestInit {
    /// @notice Tests that setShareRecipient reverts when not called by ProxyAdmin owner.
    function testFuzz_setShareRecipient_notProxyAdminOwner_reverts(
        address _caller,
        address payable _newShareRecipient
    )
        external
    {
        vm.assume(_caller != proxyAdminOwner);

        vm.expectRevert(ISuperchainRevSharesCalculator.SharesCalculator_OnlyProxyAdminOwner.selector);
        vm.prank(_caller);
        superchainRevSharesCalculator.setShareRecipient(_newShareRecipient);
    }

    /// @notice Tests that setShareRecipient updates recipient and emits event.
    function testFuzz_setShareRecipient_succeeds(address payable _newShareRecipient) external {
        vm.expectEmit(address(superchainRevSharesCalculator));
        emit ShareRecipientUpdated(shareRecipient, _newShareRecipient);

        vm.prank(proxyAdminOwner);
        superchainRevSharesCalculator.setShareRecipient(_newShareRecipient);

        assertEq(address(superchainRevSharesCalculator.shareRecipient()), address(_newShareRecipient));
    }
}

/// @notice Tests for SuperchainRevSharesCalculator setRemainderRecipient function success cases.
contract SuperchainRevSharesCalculator_SetRemainderRecipient_Test is SuperchainRevSharesCalculator_TestInit {
    /// @notice Tests that setRemainderRecipient reverts when not called by ProxyAdmin owner.
    function testFuzz_setRemainderRecipient_notProxyAdminOwner_reverts(
        address _caller,
        address payable _newRemainderRecipient
    )
        external
    {
        vm.assume(_caller != proxyAdminOwner);

        vm.expectRevert(ISuperchainRevSharesCalculator.SharesCalculator_OnlyProxyAdminOwner.selector);
        vm.prank(_caller);
        superchainRevSharesCalculator.setRemainderRecipient(_newRemainderRecipient);
    }

    /// @notice Tests that setRemainderRecipient updates recipient and emits event.
    function testFuzz_setRemainderRecipient_succeeds(address payable _newRemainderRecipient) external {
        vm.expectEmit(address(superchainRevSharesCalculator));
        emit RemainderRecipientUpdated(remainderRecipient, _newRemainderRecipient);

        vm.prank(proxyAdminOwner);
        superchainRevSharesCalculator.setRemainderRecipient(_newRemainderRecipient);

        assertEq(address(superchainRevSharesCalculator.remainderRecipient()), address(_newRemainderRecipient));
    }
}

/// @notice Tests for SuperchainRevSharesCalculator getRecipientsAndAmounts function.
contract SuperchainRevSharesCalculator_getRecipientsAndAmounts_Test is SuperchainRevSharesCalculator_TestInit {
    /// @notice Test that getRecipientsAndAmounts reverts when gross share is calculated to be 0.
    function testFuzz_getRecipientsAndAmounts_zeroGrossShare_reverts(
        uint256 _sequencerFees,
        uint256 _baseFees,
        uint256 _operatorFees,
        uint256 _l1Fees
    )
        external
    {
        // Bound each fee to ensure total revenue < 40 to make gross share = 0
        // With GROSS_SHARE_BPS = 250 and BASIS_POINT_SCALE = 10_000:
        // grossShare = (totalRevenue * 250) / 10_000 = 0 when totalRevenue < 40
        _sequencerFees = bound(_sequencerFees, 1, 10);
        _baseFees = bound(_baseFees, 1, 10);
        _operatorFees = bound(_operatorFees, 1, 10);
        _l1Fees = bound(_l1Fees, 1, 9);

        // Verify that gross share would be 0 due to integer division
        uint256 totalRevenue = _sequencerFees + _baseFees + _operatorFees + _l1Fees;
        uint256 grossShare = (totalRevenue * uint256(superchainRevSharesCalculator.GROSS_SHARE_BPS()))
            / uint256(superchainRevSharesCalculator.BASIS_POINT_SCALE());
        assertEq(grossShare, 0, "Gross share should be 0");

        // Expect the function to revert with the correct error
        vm.expectRevert(ISuperchainRevSharesCalculator.SharesCalculator_ZeroGrossShare.selector);
        superchainRevSharesCalculator.getRecipientsAndAmounts(_sequencerFees, _baseFees, _operatorFees, _l1Fees);
    }

    /// @notice Fuzz test for cases where gross share is higher than net share.
    function testFuzz_getRecipientsAndAmounts_grossShareHigher_succeeds(
        uint256 _sequencerFees,
        uint256 _baseFees,
        uint256 _operatorFees,
        uint256 _l1Fees
    )
        external
        view
    {
        // Use smaller bounds to prevent overflow
        _sequencerFees = bound(_sequencerFees, 10000, type(uint112).max);
        _baseFees = bound(_baseFees, 10000, type(uint112).max);
        _operatorFees = bound(_operatorFees, 10000, type(uint112).max);

        // Calculate other fees (without L1 fees)
        uint256 otherFees = _sequencerFees + _baseFees + _operatorFees;

        // For gross > net: we need L1 fees to be very high relative to other fees
        // Set L1 fees to be 90% of total revenue to ensure gross > net
        uint256 minL1Fees = otherFees * 9; // L1 fees = 90% of other fees, so total = 10 * otherFees, L1 = 9 * otherFees
        _l1Fees = bound(_l1Fees, minL1Fees, minL1Fees + 10000);

        ISharesCalculator.ShareInfo[] memory result =
            superchainRevSharesCalculator.getRecipientsAndAmounts(_sequencerFees, _baseFees, _operatorFees, _l1Fees);

        // Verify structure
        assertEq(result.length, 2);
        assertEq(address(result[0].recipient), address(shareRecipient));
        assertEq(address(result[1].recipient), address(remainderRecipient));

        // Calculate expected values
        uint256 totalRevenue = _sequencerFees + _baseFees + _operatorFees + _l1Fees;
        uint256 grossShare = (totalRevenue * uint256(superchainRevSharesCalculator.GROSS_SHARE_BPS()))
            / uint256(superchainRevSharesCalculator.BASIS_POINT_SCALE());
        uint256 netRevenue = totalRevenue - _l1Fees;
        uint256 netShare = (netRevenue * uint256(superchainRevSharesCalculator.NET_SHARE_BPS()))
            / uint256(superchainRevSharesCalculator.BASIS_POINT_SCALE());

        // Verify gross share is indeed higher
        assertGt(grossShare, netShare, "Gross share should be higher than net share");

        // Verify calculations
        assertEq(result[0].amount, grossShare);
        assertEq(result[1].amount, totalRevenue - grossShare);

        // Verify total conservation
        assertEq(result[0].amount + result[1].amount, totalRevenue);
    }

    /// @notice Fuzz test for cases where net share is higher than gross share.
    function testFuzz_getRecipientsAndAmounts_netShareHigher_succeeds(
        uint256 _sequencerFees,
        uint256 _baseFees,
        uint256 _operatorFees,
        uint256 _l1Fees
    )
        external
        view
    {
        // Use smaller bounds to prevent overflow
        _sequencerFees = bound(_sequencerFees, 10000, type(uint112).max);
        _baseFees = bound(_baseFees, 10000, type(uint112).max);
        _operatorFees = bound(_operatorFees, 10000, type(uint112).max);

        // Calculate other fees (without L1 fees)
        uint256 otherFees = _sequencerFees + _baseFees + _operatorFees;

        // For net > gross: we need L1 fees to be very low relative to other fees
        // Set L1 fees to be 10% of other fees to ensure net > gross
        uint256 maxL1Fees = otherFees / 10; // L1 fees = 10% of other fees
        _l1Fees = bound(_l1Fees, 1, maxL1Fees);

        ISharesCalculator.ShareInfo[] memory result =
            superchainRevSharesCalculator.getRecipientsAndAmounts(_sequencerFees, _baseFees, _operatorFees, _l1Fees);

        // Verify structure
        assertEq(result.length, 2);
        assertEq(address(result[0].recipient), address(shareRecipient));
        assertEq(address(result[1].recipient), address(remainderRecipient));

        // Calculate expected values
        uint256 totalRevenue = _sequencerFees + _baseFees + _operatorFees + _l1Fees;
        uint256 grossShare = (totalRevenue * uint256(superchainRevSharesCalculator.GROSS_SHARE_BPS()))
            / uint256(superchainRevSharesCalculator.BASIS_POINT_SCALE());
        uint256 netRevenue = totalRevenue - _l1Fees;
        uint256 netShare = (netRevenue * uint256(superchainRevSharesCalculator.NET_SHARE_BPS()))
            / uint256(superchainRevSharesCalculator.BASIS_POINT_SCALE());

        // Verify net share is indeed higher
        assertGt(netShare, grossShare, "Net share should be higher than gross share");

        // Verify calculations
        assertEq(result[0].amount, netShare);
        assertEq(result[1].amount, totalRevenue - netShare);

        // Verify total conservation
        assertEq(result[0].amount + result[1].amount, totalRevenue);
    }

    /// @notice Comprehensive fuzz test for calculation logic.
    function testFuzz_getRecipientsAndAmounts_succeeds(
        uint256 _sequencerFees,
        uint256 _baseFees,
        uint256 _operatorFees,
        uint256 _l1Fees
    )
        external
        view
    {
        // Use uint112 to prevent overflow when adding & 10000 to prevent 0 fee revert
        _sequencerFees = bound(_sequencerFees, 10000, type(uint112).max);
        _baseFees = bound(_baseFees, 10000, type(uint112).max);
        _operatorFees = bound(_operatorFees, 10000, type(uint112).max);
        _l1Fees = bound(_l1Fees, 10000, type(uint112).max);

        ISharesCalculator.ShareInfo[] memory result =
            superchainRevSharesCalculator.getRecipientsAndAmounts(_sequencerFees, _baseFees, _operatorFees, _l1Fees);

        // Verify structure
        assertEq(result.length, 2);
        assertEq(address(result[0].recipient), address(shareRecipient));
        assertEq(address(result[1].recipient), address(remainderRecipient));

        // Calculate expected values
        uint256 totalRevenue = _sequencerFees + _baseFees + _operatorFees + _l1Fees;
        uint256 grossShare = (totalRevenue * uint256(superchainRevSharesCalculator.GROSS_SHARE_BPS()))
            / uint256(superchainRevSharesCalculator.BASIS_POINT_SCALE());
        uint256 netRevenue = totalRevenue - _l1Fees;
        uint256 netShare = (netRevenue * uint256(superchainRevSharesCalculator.NET_SHARE_BPS()))
            / uint256(superchainRevSharesCalculator.BASIS_POINT_SCALE());
        uint256 expectedShareAmount = grossShare > netShare ? grossShare : netShare;

        // Verify calculations
        assertEq(result[0].amount, expectedShareAmount);
        assertEq(result[1].amount, totalRevenue - expectedShareAmount);

        // Verify total conservation
        assertEq(result[0].amount + result[1].amount, totalRevenue);
    }
}
