// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { LegacyFeeSplitter } from "test/mocks/LegacyFeeSplitter.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IFeeSplitter } from "interfaces/L2/IFeeSplitter.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

/// @title FeeSplitter_TestInit
/// @notice Reusable test initialization for `FeeSplitter` tests.
contract FeeSplitter_TestInit is CommonTest {
    // Events
    event FeesReceived(address indexed sender, uint256 amount);
    event FeeDisbursementIntervalUpdated(uint128 oldFeeDisbursementInterval, uint128 newFeeDisbursementInterval);
    event FeesDisbursed(ISharesCalculator.ShareInfo[] shareInfo, uint256 grossRevenue);
    event SharesCalculatorUpdated(address oldSharesCalculator, address newSharesCalculator);

    // Test constants
    address internal _owner;
    address internal _defaultRevenueShareRecipient = makeAddr("RevenueShareRecipient");
    address internal _defaultRevenueRemainderRecipient = makeAddr("RemainderRecipient");
    uint128 internal _defaultFeeDisbursementInterval = 1 days;
    address internal _defaultSharesCalculator = makeAddr("SharesCalculator");

    /// @notice Test setup.
    function setUp() public virtual override {
        // Enable revenue sharing before calling parent setUp
        super.enableRevenueShare();
        super.setUp();

        // Get the owner from ProxyAdmin
        _owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();
    }

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }

    /// @notice Helper to mock fee vault calls for successful withdrawal scenarios
    function _mockFeeVaultForSuccessfulWithdrawal(address _vault, uint256 _balance) internal {
        _mockFeeVaultForSuccessfulWithdrawalWithSplitter(address(feeSplitter), _vault, _balance);
    }

    /// @notice Helper to mock fee vault calls for successful withdrawal scenarios with a splitter different from the
    /// Predeploy FeeSplitter
    function _mockFeeVaultForSuccessfulWithdrawalWithSplitter(
        address _splitter,
        address _vault,
        uint256 _balance
    )
        internal
    {
        // Deploy a simple mock vault that can transfer ETH when withdraw() is called
        MockFeeVault mockVault = new MockFeeVault(payable(address(_splitter)), 0, Types.WithdrawalNetwork.L2);
        vm.deal(address(mockVault), _balance);
        vm.etch(_vault, address(mockVault).code);
        vm.deal(_vault, _balance);
    }

    /// @notice Helper to setup standard fee vault mocks for disbursement
    function _setupStandardFeeVaultMocks(
        uint256 _sequencerBalance,
        uint256 _baseBalance,
        uint256 _l1Balance,
        uint256 _operatorBalance
    )
        internal
    {
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.SEQUENCER_FEE_WALLET, _sequencerBalance);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.BASE_FEE_VAULT, _baseBalance);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.L1_FEE_VAULT, _l1Balance);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.OPERATOR_FEE_VAULT, _operatorBalance);
    }
}

/// @title FeeSplitter_Initialize_Test
/// @notice Tests the initialization functions of the `FeeSplitter` contract.
contract FeeSplitter_Initialize_Test is FeeSplitter_TestInit {
    /// @notice Test that re-initialization fails on the already-initialized predeploy
    function test_reinitialization_reverts() public {
        // The FeeSplitter at the predeploy address is already initialized through genesis
        vm.prank(_owner);
        vm.expectRevert("Initializable: contract is already initialized");
        feeSplitter.initialize(ISharesCalculator(address(_defaultSharesCalculator)));
    }

    /// @notice Test successful initialization with proper event emission on a fresh instance
    function test_feeSplitter_initialization_succeeds() public {
        // Deploy a fresh instance for testing initialization
        address impl = address(uint160(uint256(keccak256("FeeSplitterTestImpl3"))));
        vm.etch(impl, vm.getDeployedCode("FeeSplitter.sol:FeeSplitter"));

        vm.prank(_owner);
        IFeeSplitter(payable(impl)).initialize(ISharesCalculator(address(_defaultSharesCalculator)));

        assertEq(address(IFeeSplitter(payable(impl)).sharesCalculator()), address(_defaultSharesCalculator));
        assertEq(IFeeSplitter(payable(impl)).feeDisbursementInterval(), 1 days);
    }
}

/// @title FeeSplitter_Receive_Test
/// @notice Tests the receive function of the `FeeSplitter` contract.
contract FeeSplitter_Receive_Test is FeeSplitter_TestInit {
    /// @notice Test that receive function reverts when not during disbursement
    function test_feeSplitterReceive_WhenReceiveWindowIsClosed_Reverts(address _caller, uint256 _amount) public {
        vm.deal(_caller, _amount);

        vm.prank(_caller);
        vm.expectRevert(IFeeSplitter.FeeSplitter_ReceiveWindowClosed.selector);
        payable(address(feeSplitter)).call{ value: _amount }("");
    }

    /// @notice Test receive function from non-approved vault reverts even during disbursement
    function testFuzz_feeSplitterReceive_WhenNonFeeVault_Reverts(address _caller, uint256 _amount) public {
        _amount = bound(_amount, 1 ether, type(uint256).max);
        vm.assume(_caller != Predeploys.SEQUENCER_FEE_WALLET);
        vm.assume(_caller != Predeploys.BASE_FEE_VAULT);
        vm.assume(_caller != Predeploys.OPERATOR_FEE_VAULT);
        vm.assume(_caller != Predeploys.L1_FEE_VAULT);
        vm.assume(_caller != address(0));

        // Mock the _isTransientDisbursing() function to return true
        // This allows us to test the sender validation logic
        vm.mockCall(address(feeSplitter), abi.encodeWithSignature("_isTransientDisbursing()"), abi.encode(true));

        // Setup disbursement conditions but expect revert from non-approved sender
        vm.deal(_caller, _amount);

        vm.prank(_caller);
        // Now we test the actual sender validation
        vm.expectRevert(IFeeSplitter.FeeSplitter_SenderNotApprovedVault.selector);
        payable(address(feeSplitter)).call{ value: _amount }("");
    }

    /// @notice Test receive function works during disbursement from SequencerFeeVault
    function test_feeSplitterReceive_SequencerFeeVault_Succeeds(uint256 _amount) public {
        _amount = bound(_amount, 1 ether, type(uint256).max);

        // Setup mocks - only sequencer vault has balance
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.SEQUENCER_FEE_WALLET, _amount);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.BASE_FEE_VAULT, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.L1_FEE_VAULT, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.OPERATOR_FEE_VAULT, 0);

        // Mock shares calculator to return valid shares
        ISharesCalculator.ShareInfo[] memory shareInfo = new ISharesCalculator.ShareInfo[](1);
        shareInfo[0] = ISharesCalculator.ShareInfo(payable(_defaultRevenueShareRecipient), _amount);

        // Get the actual shares calculator from the FeeSplitter
        address actualSharesCalculator = address(feeSplitter.sharesCalculator());
        vm.mockCall(
            actualSharesCalculator,
            abi.encodeCall(ISharesCalculator.getRecipientsAndAmounts, (_amount, 0, 0, 0)),
            abi.encode(shareInfo)
        );

        // Fast forward time to allow disbursement
        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        // Call disburseFees - this will trigger the receive function during withdrawal
        feeSplitter.disburseFees();

        // Verify the recipient got the funds (proves receive function worked)
        assertEq(address(_defaultRevenueShareRecipient).balance, _amount);
        assertEq(address(feeSplitter).balance, 0);
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);
    }

    /// @notice Test receive function works during disbursement from BaseFeeVault
    function test_feeSplitterReceive_BaseFeeVault_Succeeds(uint256 _amount) public {
        _amount = bound(_amount, 1 ether, type(uint256).max);

        // Setup mocks - only sequencer vault has balance
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.SEQUENCER_FEE_WALLET, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.BASE_FEE_VAULT, _amount);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.L1_FEE_VAULT, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.OPERATOR_FEE_VAULT, 0);

        // Mock shares calculator to return valid shares
        ISharesCalculator.ShareInfo[] memory shareInfo = new ISharesCalculator.ShareInfo[](1);
        shareInfo[0] = ISharesCalculator.ShareInfo(payable(_defaultRevenueShareRecipient), _amount);

        // Get the actual shares calculator from the FeeSplitter
        address actualSharesCalculator = address(feeSplitter.sharesCalculator());
        vm.mockCall(
            actualSharesCalculator,
            abi.encodeCall(ISharesCalculator.getRecipientsAndAmounts, (0, _amount, 0, 0)),
            abi.encode(shareInfo)
        );

        // Fast forward time to allow disbursement
        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        // Call disburseFees - this will trigger the receive function during withdrawal
        feeSplitter.disburseFees();

        // Verify the recipient got the funds (proves receive function worked)
        assertEq(address(_defaultRevenueShareRecipient).balance, _amount);
        assertEq(address(feeSplitter).balance, 0);
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);
    }

    /// @notice Test receive function works during disbursement from L1FeeVault
    function test_feeSplitterReceive_L1FeeVault_Succeeds(uint256 _amount) public {
        _amount = bound(_amount, 1 ether, type(uint256).max);

        // Setup mocks - only sequencer vault has balance
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.SEQUENCER_FEE_WALLET, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.BASE_FEE_VAULT, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.L1_FEE_VAULT, _amount);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.OPERATOR_FEE_VAULT, 0);

        // Mock shares calculator to return valid shares
        ISharesCalculator.ShareInfo[] memory shareInfo = new ISharesCalculator.ShareInfo[](1);
        shareInfo[0] = ISharesCalculator.ShareInfo(payable(_defaultRevenueShareRecipient), _amount);

        // Get the actual shares calculator from the FeeSplitter
        address actualSharesCalculator = address(feeSplitter.sharesCalculator());
        vm.mockCall(
            actualSharesCalculator,
            abi.encodeCall(ISharesCalculator.getRecipientsAndAmounts, (0, 0, 0, _amount)),
            abi.encode(shareInfo)
        );

        // Fast forward time to allow disbursement
        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        // Call disburseFees - this will trigger the receive function during withdrawal
        feeSplitter.disburseFees();

        // Verify the recipient got the funds (proves receive function worked)
        assertEq(address(_defaultRevenueShareRecipient).balance, _amount);
        assertEq(address(feeSplitter).balance, 0);
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);
    }

    /// @notice Test receive function works during disbursement from OperatorFeeVault
    function test_feeSplitterReceive_OperatorFeeVault_Succeeds(uint256 _amount) public {
        _amount = bound(_amount, 1 ether, type(uint256).max);

        // Setup mocks - only sequencer vault has balance
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.SEQUENCER_FEE_WALLET, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.BASE_FEE_VAULT, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.L1_FEE_VAULT, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.OPERATOR_FEE_VAULT, _amount);

        // Mock shares calculator to return valid shares
        ISharesCalculator.ShareInfo[] memory shareInfo = new ISharesCalculator.ShareInfo[](1);
        shareInfo[0] = ISharesCalculator.ShareInfo(payable(_defaultRevenueShareRecipient), _amount);

        // Get the actual shares calculator from the FeeSplitter
        address actualSharesCalculator = address(feeSplitter.sharesCalculator());
        vm.mockCall(
            actualSharesCalculator,
            abi.encodeCall(ISharesCalculator.getRecipientsAndAmounts, (0, 0, _amount, 0)),
            abi.encode(shareInfo)
        );

        // Fast forward time to allow disbursement
        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        // Call disburseFees - this will trigger the receive function during withdrawal
        feeSplitter.disburseFees();

        // Verify the recipient got the funds (proves receive function worked)
        assertEq(address(_defaultRevenueShareRecipient).balance, _amount);
        assertEq(address(feeSplitter).balance, 0);
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);
    }
}

/// @title FeeSplitter_DisburseFees_Test
/// @notice Tests the disburseFees function of the `FeeSplitter` contract.
contract FeeSplitter_DisburseFees_Test is FeeSplitter_TestInit {
    /// @notice Test disburseFees reverts when interval not reached
    function test_feeSplitterDisburseFees_WhenIntervalNotReached_Reverts() public {
        vm.prank(_owner);
        feeSplitter.setFeeDisbursementInterval(48 hours);

        vm.expectRevert(IFeeSplitter.FeeSplitter_DisbursementIntervalNotReached.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test disburseFees reverts when no fees collected
    function test_feeSplitterDisburseFees_WhenNoFeesCollected_Reverts() public {
        _setupStandardFeeVaultMocks(0, 0, 0, 0);

        vm.warp(block.timestamp + 25 hours);
        vm.expectRevert(IFeeSplitter.FeeSplitter_NoFeesCollected.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test disburseFees fails when fee vault has wrong withdrawal network
    function test_feeSplitterDisburseFees_WhenFeeVaultWrongNetwork_Reverts() public {
        // Mock fee vault with L1 withdrawal network (invalid)
        vm.mockCall(
            Predeploys.SEQUENCER_FEE_WALLET,
            abi.encodeCall(IFeeVault.withdrawalNetwork, ()),
            abi.encode(Types.WithdrawalNetwork.L1)
        );

        vm.warp(block.timestamp + 25 hours);
        vm.expectRevert(IFeeSplitter.FeeSplitter_FeeVaultMustWithdrawToL2.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test disburseFees fails when fee vault has wrong recipient
    function test_feeSplitterDisburseFees_WhenFeeVaultWrongRecipient_Reverts() public {
        // Mock fee vault with wrong recipient
        vm.mockCall(
            Predeploys.SEQUENCER_FEE_WALLET,
            abi.encodeCall(IFeeVault.withdrawalNetwork, ()),
            abi.encode(Types.WithdrawalNetwork.L2)
        );
        vm.mockCall(
            Predeploys.SEQUENCER_FEE_WALLET, abi.encodeCall(IFeeVault.recipient, ()), abi.encode(address(0x123))
        );

        vm.warp(block.timestamp + 25 hours);
        vm.expectRevert(IFeeSplitter.FeeSplitter_FeeVaultMustWithdrawToFeeSplitter.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test successful fee disbursement with fixed amounts
    function test_feeSplitterDisburseFees_succeeds() public {
        uint256 _sequencerAmount = 2 ether;
        uint256 _baseAmount = 3 ether;
        uint256 _l1Amount = 1 ether;
        uint256 _operatorAmount = 4 ether;

        _setupStandardFeeVaultMocks(_sequencerAmount, _baseAmount, _l1Amount, _operatorAmount);

        // Calculate expected gross revenue
        uint256 expectedGrossRevenue = _sequencerAmount + _baseAmount + _l1Amount + _operatorAmount;

        // Setup mock shares calculator to return 50/50 split
        uint256 halfGrossRevenue = expectedGrossRevenue / 2;
        ISharesCalculator.ShareInfo[] memory expectedShareInfo = new ISharesCalculator.ShareInfo[](2);
        expectedShareInfo[0] = ISharesCalculator.ShareInfo(payable(_defaultRevenueShareRecipient), halfGrossRevenue);
        expectedShareInfo[1] = ISharesCalculator.ShareInfo(
            payable(_defaultRevenueRemainderRecipient), expectedGrossRevenue - halfGrossRevenue
        );

        // Get the actual shares calculator from the FeeSplitter
        address actualSharesCalculator = address(feeSplitter.sharesCalculator());
        vm.mockCall(
            actualSharesCalculator,
            abi.encodeCall(
                ISharesCalculator.getRecipientsAndAmounts, (_sequencerAmount, _baseAmount, _operatorAmount, _l1Amount)
            ),
            abi.encode(expectedShareInfo)
        );

        // Fast forward time to allow disbursement
        vm.warp(block.timestamp + 25 hours);

        // Store initial balances
        uint256 revenueShareRecipientBalanceBefore = address(_defaultRevenueShareRecipient).balance;
        uint256 revenueRemainderRecipientBalanceBefore = address(_defaultRevenueRemainderRecipient).balance;

        // Call disburseFees
        feeSplitter.disburseFees();

        // Verify the last disbursement time was updated
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);

        // Verify recipients received their shares
        assertEq(address(_defaultRevenueShareRecipient).balance, revenueShareRecipientBalanceBefore + halfGrossRevenue);
        assertEq(
            address(_defaultRevenueRemainderRecipient).balance,
            revenueRemainderRecipientBalanceBefore + (expectedGrossRevenue - halfGrossRevenue)
        );

        // Verify the fee vaults have no balance
        assertEq(address(Predeploys.SEQUENCER_FEE_WALLET).balance, 0);
        assertEq(address(Predeploys.BASE_FEE_VAULT).balance, 0);
        assertEq(address(Predeploys.L1_FEE_VAULT).balance, 0);
        assertEq(address(Predeploys.OPERATOR_FEE_VAULT).balance, 0);

        // Verify the fee splitter has no balance
        assertEq(address(feeSplitter).balance, 0);
    }
}

/// @title FeeSplitter_SetSharesCalculator_Test
/// @notice Tests the setSharesCalculator function of the `FeeSplitter` contract.
contract FeeSplitter_SetSharesCalculator_Test is FeeSplitter_TestInit {
    /// @notice Test setSharesCalculator reverts when caller is not owner
    function testFuzz_feeSplitterSetSharesCalculator_WhenNotOwner_Reverts(address _caller) public {
        vm.assume(_caller != _owner);

        vm.prank(_caller);
        vm.expectRevert(IFeeSplitter.FeeSplitter_OnlyProxyAdminOwner.selector);
        feeSplitter.setSharesCalculator(ISharesCalculator(address(0x123)));
    }

    /// @notice Test setSharesCalculator reverts with zero address
    function test_feeSplitterSetSharesCalculator_WhenZeroAddress_Reverts() public {
        vm.prank(_owner);
        vm.expectRevert(IFeeSplitter.FeeSplitter_SharesCalculatorCannotBeZero.selector);
        feeSplitter.setSharesCalculator(ISharesCalculator(address(0)));
    }

    /// @notice Test successful setSharesCalculator
    function test_feeSplitterSetSharesCalculator_succeeds(address _newSharesCalculator) public {
        vm.assume(_newSharesCalculator != address(0));

        vm.expectEmit(address(feeSplitter));
        emit SharesCalculatorUpdated(address(feeSplitter.sharesCalculator()), _newSharesCalculator);

        vm.prank(_owner);
        feeSplitter.setSharesCalculator(ISharesCalculator(_newSharesCalculator));

        assertEq(address(feeSplitter.sharesCalculator()), _newSharesCalculator);
    }
}

/// @title FeeSplitter_SetFeeDisbursementInterval_Test
/// @notice Tests the setFeeDisbursementInterval function of the `FeeSplitter` contract.
contract FeeSplitter_SetFeeDisbursementInterval_Test is FeeSplitter_TestInit {
    /// @notice Test setFeeDisbursementInterval reverts when caller is not owner
    function testFuzz_feeSplitterSetFeeDisbursementInterval_WhenNotOwner_Reverts(address _caller) public {
        vm.assume(_caller != _owner);

        vm.prank(_caller);
        vm.expectRevert(IFeeSplitter.FeeSplitter_OnlyProxyAdminOwner.selector);
        feeSplitter.setFeeDisbursementInterval(48 hours);
    }

    /// @notice Test setFeeDisbursementInterval reverts when interval is too long
    function testFuzz_feeSplitterSetFeeDisbursementInterval_WhenIntervalTooLong_Reverts(uint256 _disbursementInterval)
        public
    {
        _disbursementInterval = bound(_disbursementInterval, 365 days + 1, type(uint128).max);

        vm.prank(_owner);
        vm.expectRevert(IFeeSplitter.FeeSplitter_ExceedsMaxFeeDisbursementTime.selector);
        feeSplitter.setFeeDisbursementInterval(uint128(_disbursementInterval));
    }

    /// @notice Test successful setFeeDisbursementInterval
    function testFuzz_feeSplitterSetFeeDisbursementInterval_succeeds(uint128 _newInterval) public {
        _newInterval = uint128(bound(_newInterval, 1, 365 days));

        vm.expectEmit(address(feeSplitter));
        emit FeeDisbursementIntervalUpdated(feeSplitter.feeDisbursementInterval(), _newInterval);

        vm.prank(_owner);
        feeSplitter.setFeeDisbursementInterval(_newInterval);

        assertEq(feeSplitter.feeDisbursementInterval(), _newInterval);
    }
}

/// @title FeeSplitter_DisburseFees_TestFail
/// @notice Test failure scenario where vaults have insufficient balance for withdrawal
contract FeeSplitter_DisburseFees_TestFail is FeeSplitter_TestInit {
    /// @notice Helper to mock fee vault with specific minimum withdrawal amount
    function _setFeeVaultData(address _vault, uint256 _balance, uint256 _minWithdrawal) internal {
        MockFeeVault mockVault =
            new MockFeeVault(payable(address(feeSplitter)), _minWithdrawal, Types.WithdrawalNetwork.L2);
        vm.deal(address(mockVault), _balance);
        vm.etch(_vault, address(mockVault).code);
        vm.deal(_vault, _balance);
    }

    /// @notice Fuzz test that a vault with balance below minimum causes entire disbursement to revert
    function test_disburseFees_vaultBelowMinimum_Reverts(uint256 _minWithdrawalAmount, uint256 _vaultIndex) public {
        // If uint256, the test will revert due to ETH transfer overflow
        _minWithdrawalAmount = bound(_minWithdrawalAmount, 1, type(uint128).max);
        _vaultIndex = bound(_vaultIndex, 0, 3); // 0-3 for the 4 vaults

        // Calculate vault balances: one vault will have insufficient balance
        uint256 insufficientBalance = _minWithdrawalAmount - 1;
        uint256 sufficientBalance = _minWithdrawalAmount;

        address[4] memory vaults = [
            Predeploys.SEQUENCER_FEE_WALLET,
            Predeploys.BASE_FEE_VAULT,
            Predeploys.L1_FEE_VAULT,
            Predeploys.OPERATOR_FEE_VAULT
        ];

        // Setup all vaults with sufficient balance first
        for (uint256 i = 0; i < 4; i++) {
            _setFeeVaultData(vaults[i], sufficientBalance, _minWithdrawalAmount);
        }

        // Override the selected vault with insufficient balance
        _setFeeVaultData(vaults[_vaultIndex], insufficientBalance, _minWithdrawalAmount);

        vm.warp(block.timestamp + 25 hours);

        // The entire disbursement should revert because one vault doesn't meet its minimum
        vm.expectRevert("FeeVault: withdrawal amount must be greater than minimum withdrawal amount");
        feeSplitter.disburseFees();

        // Verify no funds were moved (all vaults retain their original balance)
        for (uint256 i = 0; i < 4; i++) {
            uint256 expectedBalance = (i == _vaultIndex) ? insufficientBalance : sufficientBalance;
            assertEq(address(vaults[i]).balance, expectedBalance);
        }
    }
}

contract LegacyFeeSplitter_DisburseFees_Test is FeeSplitter_TestInit {
    LegacyFeeSplitter public legacyFeeSplitter;

    function setUp() public override {
        super.setUp();

        legacyFeeSplitter = new LegacyFeeSplitter();

        // Setup the legacy splitter as the recipient in the vaults
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        vm.startPrank(owner);
        IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).setRecipient(address(legacyFeeSplitter));
        IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).setRecipient(address(legacyFeeSplitter));
        IFeeVault(payable(Predeploys.L1_FEE_VAULT)).setRecipient(address(legacyFeeSplitter));
        IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).setRecipient(address(legacyFeeSplitter));
        vm.stopPrank();
    }

    function test_legacyFeeSplitterDisburseFees_succeeds(
        uint256 _sequencerBalance,
        uint256 _baseBalance,
        uint256 _l1Balance,
        uint256 _operatorBalance
    )
        public
    {
        _sequencerBalance = bound(
            _sequencerBalance,
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).minWithdrawalAmount(),
            type(uint128).max
        );

        _baseBalance =
            bound(_baseBalance, IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).minWithdrawalAmount(), type(uint128).max);

        _l1Balance =
            bound(_l1Balance, IFeeVault(payable(Predeploys.L1_FEE_VAULT)).minWithdrawalAmount(), type(uint128).max);

        _operatorBalance = bound(
            _operatorBalance, IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).minWithdrawalAmount(), type(uint128).max
        );

        // Setup mock fee vaults
        _mockFeeVaultForSuccessfulWithdrawalWithSplitter(
            address(legacyFeeSplitter), Predeploys.SEQUENCER_FEE_WALLET, uint256(_sequencerBalance)
        );
        _mockFeeVaultForSuccessfulWithdrawalWithSplitter(
            address(legacyFeeSplitter), Predeploys.BASE_FEE_VAULT, uint256(_baseBalance)
        );
        _mockFeeVaultForSuccessfulWithdrawalWithSplitter(
            address(legacyFeeSplitter), Predeploys.L1_FEE_VAULT, uint256(_l1Balance)
        );
        _mockFeeVaultForSuccessfulWithdrawalWithSplitter(
            address(legacyFeeSplitter), Predeploys.OPERATOR_FEE_VAULT, uint256(_operatorBalance)
        );

        assertEq(address(legacyFeeSplitter).balance, 0);
        legacyFeeSplitter.disburseFees();
        assertEq(address(legacyFeeSplitter).balance, _sequencerBalance + _baseBalance + _l1Balance + _operatorBalance);
    }
}

/// @notice Simple mock FeeVault for testing that actually transfers ETH
contract MockFeeVault {
    uint256 public immutable MIN_WITHDRAWAL_AMOUNT;
    address public immutable RECIPIENT;
    Types.WithdrawalNetwork public immutable WITHDRAWAL_NETWORK;

    event Withdrawal(uint256 value, address to, address from);
    event Withdrawal(uint256 value, address to, address from, Types.WithdrawalNetwork withdrawalNetwork);

    constructor(address payable _recipient, uint256 _minWithdrawalAmount, Types.WithdrawalNetwork _withdrawalNetwork) {
        RECIPIENT = _recipient;
        MIN_WITHDRAWAL_AMOUNT = _minWithdrawalAmount;
        WITHDRAWAL_NETWORK = _withdrawalNetwork;
    }

    receive() external payable { }

    function withdrawalNetwork() external view returns (Types.WithdrawalNetwork) {
        return WITHDRAWAL_NETWORK;
    }

    function minWithdrawalAmount() external view returns (uint256) {
        return MIN_WITHDRAWAL_AMOUNT;
    }

    function recipient() external view returns (address) {
        return RECIPIENT;
    }

    function withdraw() external returns (uint256) {
        require(
            address(this).balance >= MIN_WITHDRAWAL_AMOUNT,
            "FeeVault: withdrawal amount must be greater than minimum withdrawal amount"
        );

        uint256 value = address(this).balance;

        emit Withdrawal(value, RECIPIENT, msg.sender);
        emit Withdrawal(value, RECIPIENT, msg.sender, WITHDRAWAL_NETWORK);

        if (WITHDRAWAL_NETWORK == Types.WithdrawalNetwork.L2) {
            (bool success,) = RECIPIENT.call{ value: value }("");
            require(success, "FeeVault: failed to send ETH to L2 fee recipient");
        }

        return value;
    }
}
