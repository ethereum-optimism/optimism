// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Mocks
import { MockFeeVault } from "test/mocks/MockFeeVault.sol";
import { MaliciousMockFeeVault } from "test/mocks/MaliciousMockFeeVault.sol";
import { RevertingRecipient } from "test/mocks/RevertingRecipient.sol";
import { ReentrantMockFeeVault } from "test/mocks/ReentrantMockFeeVault.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Interfaces
import { IFeeSplitter } from "interfaces/L2/IFeeSplitter.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

/// @title FeeSplitter_TestInit
/// @notice Reusable test initialization for `FeeSplitter` tests.
contract FeeSplitter_TestInit is CommonTest {
    // Events
    event FeesReceived(address indexed sender, uint256 amount, uint256 newBalance);
    event FeeDisbursementIntervalUpdated(uint128 oldFeeDisbursementInterval, uint128 newFeeDisbursementInterval);
    event FeesDisbursed(ISharesCalculator.ShareInfo[] shareInfo, uint256 grossRevenue);
    event SharesCalculatorUpdated(address oldSharesCalculator, address newSharesCalculator);

    // Test constants
    address internal _owner;
    address internal _defaultRevenueShareRecipient = makeAddr("RevenueShareRecipient");
    address internal _defaultRevenueRemainderRecipient = makeAddr("RemainderRecipient");
    uint128 internal _defaultFeeDisbursementInterval = 1 days;
    address internal _defaultSharesCalculator = makeAddr("SharesCalculator");
    address[4] internal _feeVaults;
    bytes32 internal constant _FEE_SPLITTER_DISBURSING_ADDRESS_SLOT =
        0x21346dddac42cc163a6523eefc19df981df7352c870dc3b0b17a6a92fc6fe813;

    /// @notice Test setup.
    function setUp() public virtual override {
        // Resolve features and skip whole test suite if custom gas token is enabled
        resolveFeaturesFromEnv();
        skipIfDevFeatureEnabled(DevFeatures.CUSTOM_GAS_TOKEN);

        // Enable revenue sharing before calling parent setUp
        super.enableRevenueShare();
        super.setUp();

        // Get the owner from ProxyAdmin
        _owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        // Initialize fee vaults array
        _feeVaults[0] = Predeploys.SEQUENCER_FEE_WALLET;
        _feeVaults[1] = Predeploys.BASE_FEE_VAULT;
        _feeVaults[2] = Predeploys.L1_FEE_VAULT;
        _feeVaults[3] = Predeploys.OPERATOR_FEE_VAULT;
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
    event Initialized(uint8 version);

    /// @notice Test that re-initialization fails on the already-initialized predeploy
    function test_feeSplitter_reinitialization_reverts() public {
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
        assertEq(IFeeSplitter(payable(impl)).lastDisbursementTime(), block.timestamp);
    }

    /// @notice Test that the implementation contract disables initializers in the constructor
    function test_feeSplitterImplementation_constructorDisablesInitializers_succeeds() public {
        bytes memory creationCode = vm.getCode("FeeSplitter.sol:FeeSplitter");
        address implementation;

        // Expect the Initialized event to be emitted
        vm.expectEmit(true, true, true, true);
        emit Initialized(type(uint8).max);

        // Deploy the implementation contract
        assembly {
            implementation := create(0, add(creationCode, 0x20), mload(creationCode))
        }

        // Verify the implementation contract is not zero address
        assertTrue(implementation != address(0));

        // Verify re-initialization fails
        vm.expectRevert("Initializable: contract is already initialized");
        IFeeSplitter(payable(implementation)).initialize(ISharesCalculator(address(_defaultSharesCalculator)));
    }
}

/// @title FeeSplitter_Receive_Test
/// @notice Tests the receive function of the `FeeSplitter` contract.
contract FeeSplitter_Receive_Test is FeeSplitter_TestInit {
    /// @notice Test that receive function reverts when sender is not an approved vault
    function testFuzz_feeSplitterReceive_whenNotApprovedVault_reverts(address _caller, uint256 _amount) public {
        vm.assume(_caller != address(0));
        vm.assume(_caller != Predeploys.SEQUENCER_FEE_WALLET);
        vm.assume(_caller != Predeploys.BASE_FEE_VAULT);
        vm.assume(_caller != Predeploys.OPERATOR_FEE_VAULT);
        vm.assume(_caller != Predeploys.L1_FEE_VAULT);
        vm.deal(_caller, _amount);

        vm.prank(_caller);
        vm.expectRevert(IFeeSplitter.FeeSplitter_SenderNotCurrentVault.selector);
        (bool revertsAsExpected,) = payable(address(feeSplitter)).call{ value: _amount }("");
        assertTrue(revertsAsExpected, "FeeSplitter_Test: call did not revert");
    }

    /// @notice Test receive function from non-approved vault reverts even during disbursement
    function testFuzz_feeSplitterReceive_whenNonFeeVault_reverts(address _caller, uint256 _amount) public {
        vm.assume(_caller != address(0));
        vm.assume(_caller != Predeploys.SEQUENCER_FEE_WALLET);
        vm.assume(_caller != Predeploys.BASE_FEE_VAULT);
        vm.assume(_caller != Predeploys.OPERATOR_FEE_VAULT);
        vm.assume(_caller != Predeploys.L1_FEE_VAULT);

        // Setup disbursement conditions but expect revert from non-approved sender
        vm.deal(_caller, _amount);
        vm.startPrank(_caller);

        // Now we test the actual sender validation
        vm.expectRevert(IFeeSplitter.FeeSplitter_SenderNotCurrentVault.selector);
        (bool revertsAsExpected,) = payable(address(feeSplitter)).call{ value: _amount }("");
        assertTrue(revertsAsExpected, "FeeSplitter_Test: call did not revert");
    }

    /// @notice Test receive function works during disbursement from SequencerFeeVault
    function testFuzz_feeSplitterReceive_sequencerFeeVault_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 1, type(uint256).max);

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

        vm.expectEmit(true, true, true, true);
        emit FeesReceived(Predeploys.SEQUENCER_FEE_WALLET, _amount, _amount);

        // Call disburseFees - this will trigger the receive function during withdrawal
        feeSplitter.disburseFees();

        // Verify the recipient got the funds (proves receive function worked)
        assertEq(address(_defaultRevenueShareRecipient).balance, _amount);
        assertEq(address(feeSplitter).balance, 0);
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);
    }

    /// @notice Test receive function works during disbursement from BaseFeeVault
    function testFuzz_feeSplitterReceive_baseFeeVault_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 1, type(uint256).max);

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

        vm.expectEmit(true, true, true, true);
        emit FeesReceived(Predeploys.BASE_FEE_VAULT, _amount, _amount);

        // Call disburseFees - this will trigger the receive function during withdrawal
        feeSplitter.disburseFees();

        // Verify the recipient got the funds (proves receive function worked)
        assertEq(address(_defaultRevenueShareRecipient).balance, _amount);
        assertEq(address(feeSplitter).balance, 0);
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);
    }

    /// @notice Test receive function works during disbursement from L1FeeVault
    function testFuzz_feeSplitterReceive_l1FeeVault_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 1, type(uint256).max);

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

        vm.expectEmit(true, true, true, true);
        emit FeesReceived(Predeploys.L1_FEE_VAULT, _amount, _amount);

        // Call disburseFees - this will trigger the receive function during withdrawal
        feeSplitter.disburseFees();

        // Verify the recipient got the funds (proves receive function worked)
        assertEq(address(_defaultRevenueShareRecipient).balance, _amount);
        assertEq(address(feeSplitter).balance, 0);
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);
    }

    /// @notice Test receive function works during disbursement from OperatorFeeVault
    function testFuzz_feeSplitterReceive_operatorFeeVault_succeeds(uint256 _amount) public {
        _amount = bound(_amount, 1, type(uint256).max);

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

        vm.expectEmit(true, true, true, true);
        emit FeesReceived(Predeploys.OPERATOR_FEE_VAULT, _amount, _amount);

        // Call disburseFees - this will trigger the receive function during withdrawal
        feeSplitter.disburseFees();

        // Verify the recipient got the funds (proves receive function worked)
        assertEq(address(_defaultRevenueShareRecipient).balance, _amount);
        assertEq(address(feeSplitter).balance, 0);
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);
    }

    /// @notice Test that a malicious vault cannot trigger withdrawals from other vaults during its own withdrawal.
    ///         This test demonstrates that the stricter receive() validation (checking the specific vault address)
    ///         prevents a re-entrancy attack where one vault tries to indirectly call withdraw() on a different vault.
    function test_feeSplitterReceive_reentrantVaultAttack_reverts() public {
        uint256 sequencerAmount = 1 ether;

        // Setup SEQUENCER_FEE_WALLET as a malicious vault that will try to trigger BASE_FEE_VAULT withdrawal
        ReentrantMockFeeVault maliciousVault = new ReentrantMockFeeVault(
            payable(address(feeSplitter)), sequencerAmount, payable(Predeploys.BASE_FEE_VAULT)
        );

        vm.etch(Predeploys.SEQUENCER_FEE_WALLET, address(maliciousVault).code);
        vm.deal(Predeploys.SEQUENCER_FEE_WALLET, sequencerAmount);

        // Fast forward time
        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        // Expect the disbursement to revert with the MockFeeVault error message.
        // The flow is:
        // 1. FeeSplitter:disburseFees() is called
        // 2. Splitter triggers withdrawal from SEQUENCER_FEE_WALLET (etched with malicious code)
        // 3. SEQUENCER_FEE_WALLET sends ETH to FeeSplitter and triggers withdrawal from BASE_FEE_VAULT
        // 4. BASE_FEE_VAULT's withdraw() reverts with the FeeVault's error message
        vm.expectRevert("FeeVault: failed to send ETH to L2 fee recipient");
        feeSplitter.disburseFees();
    }
}

/// @title FeeSplitter_DisburseFees_Test
/// @notice Tests the disburseFees function of the `FeeSplitter` contract.
contract FeeSplitter_DisburseFees_Test is FeeSplitter_TestInit {
    /// @notice Test disburseFees reverts when interval not reached
    function test_feeSplitterDisburseFees_whenIntervalNotReached_reverts() public {
        vm.prank(_owner);
        feeSplitter.setFeeDisbursementInterval(48 hours);

        vm.expectRevert(IFeeSplitter.FeeSplitter_DisbursementIntervalNotReached.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test disburseFees reverts when no fees collected
    function test_feeSplitterDisburseFees_whenNoFeesCollected_reverts() public {
        _setupStandardFeeVaultMocks(0, 0, 0, 0);

        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);
        vm.expectRevert(IFeeSplitter.FeeSplitter_NoFeesCollected.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test disburseFees fails when fee vault has wrong withdrawal network
    function test_feeSplitterDisburseFees_whenFeeVaultWrongNetwork_reverts() public {
        // Mock fee vault with L1 withdrawal network (invalid)
        vm.mockCall(
            Predeploys.SEQUENCER_FEE_WALLET,
            abi.encodeCall(IFeeVault.withdrawalNetwork, ()),
            abi.encode(Types.WithdrawalNetwork.L1)
        );

        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);
        vm.expectRevert(IFeeSplitter.FeeSplitter_FeeVaultMustWithdrawToL2.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test disburseFees fails when fee vault has wrong recipient
    function test_feeSplitterDisburseFees_whenFeeVaultWrongRecipient_reverts() public {
        // Mock fee vault with wrong recipient
        vm.mockCall(
            Predeploys.SEQUENCER_FEE_WALLET,
            abi.encodeCall(IFeeVault.withdrawalNetwork, ()),
            abi.encode(Types.WithdrawalNetwork.L2)
        );
        vm.mockCall(
            Predeploys.SEQUENCER_FEE_WALLET, abi.encodeCall(IFeeVault.recipient, ()), abi.encode(address(0x123))
        );

        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);
        vm.expectRevert(IFeeSplitter.FeeSplitter_FeeVaultMustWithdrawToFeeSplitter.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test disburseFees reverts when fee vault withdrawal amount does not match the expected amount
    function testFuzz_feeSplitterDisburseFees_whenFeeVaultWithdrawalAmountMismatch_reverts(
        uint256 _actualTransferAmount,
        uint256 _claimedWithdrawalAmount
    )
        public
    {
        vm.assume(_actualTransferAmount != _claimedWithdrawalAmount);

        // Create a malicious mock vault that lies about withdrawal amount
        MaliciousMockFeeVault maliciousVault =
            new MaliciousMockFeeVault(payable(address(feeSplitter)), _actualTransferAmount, _claimedWithdrawalAmount);
        vm.deal(address(maliciousVault), _actualTransferAmount);

        // Replace SEQUENCER_FEE_WALLET with the malicious vault
        vm.etch(Predeploys.SEQUENCER_FEE_WALLET, address(maliciousVault).code);
        vm.deal(Predeploys.SEQUENCER_FEE_WALLET, _actualTransferAmount);

        // Setup other vaults normally with zero balance
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.BASE_FEE_VAULT, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.L1_FEE_VAULT, 0);
        _mockFeeVaultForSuccessfulWithdrawal(Predeploys.OPERATOR_FEE_VAULT, 0);

        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);
        vm.expectRevert(IFeeSplitter.FeeSplitter_FeeVaultWithdrawalAmountMismatch.selector);
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
        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

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

    /// @notice Test disburseFees reverts when shares calculator returns an empty array
    function test_feeSplitterDisburseFees_whenSharesInfoEmpty_reverts() public {
        uint256 _sequencerAmount = 2 ether;
        _setupStandardFeeVaultMocks(_sequencerAmount, 0, 0, 0);

        address actualSharesCalculator = address(feeSplitter.sharesCalculator());
        ISharesCalculator.ShareInfo[] memory emptyShareInfo = new ISharesCalculator.ShareInfo[](0);
        vm.mockCall(
            actualSharesCalculator,
            abi.encodeCall(ISharesCalculator.getRecipientsAndAmounts, (_sequencerAmount, 0, 0, 0)),
            abi.encode(emptyShareInfo)
        );

        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        vm.expectRevert(IFeeSplitter.FeeSplitter_FeeShareInfoEmpty.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test disburseFees reverts when sending to a recipient fails
    function test_feeSplitterDisburseFees_whenSendingFails_reverts() public {
        uint256 _sequencerAmount = 1 ether;
        _setupStandardFeeVaultMocks(_sequencerAmount, 0, 0, 0);

        address revertingRecipient = address(new RevertingRecipient());
        ISharesCalculator.ShareInfo[] memory shareInfo = new ISharesCalculator.ShareInfo[](1);
        shareInfo[0] = ISharesCalculator.ShareInfo(payable(revertingRecipient), _sequencerAmount);

        address actualSharesCalculator = address(feeSplitter.sharesCalculator());
        vm.mockCall(
            actualSharesCalculator,
            abi.encodeCall(ISharesCalculator.getRecipientsAndAmounts, (_sequencerAmount, 0, 0, 0)),
            abi.encode(shareInfo)
        );

        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        vm.expectRevert(IFeeSplitter.FeeSplitter_FailedToSendToRevenueShareRecipient.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Test disburseFees reverts when total shares do not match gross revenue
    function test_feeSplitterDisburseFees_whenSharesMalformed_reverts() public {
        uint256 _sequencerAmount = 1 ether;
        _setupStandardFeeVaultMocks(_sequencerAmount, 0, 0, 0);

        ISharesCalculator.ShareInfo[] memory shareInfo = new ISharesCalculator.ShareInfo[](1);
        shareInfo[0] = ISharesCalculator.ShareInfo(payable(_defaultRevenueShareRecipient), _sequencerAmount - 1);

        address actualSharesCalculator = address(feeSplitter.sharesCalculator());
        vm.mockCall(
            actualSharesCalculator,
            abi.encodeCall(ISharesCalculator.getRecipientsAndAmounts, (_sequencerAmount, 0, 0, 0)),
            abi.encode(shareInfo)
        );

        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        vm.expectRevert(IFeeSplitter.FeeSplitter_SharesCalculatorMalformedOutput.selector);
        feeSplitter.disburseFees();
    }

    /// @notice Fuzz test that a vault with balance below minimum causes entire disbursement to revert
    function testFuzz_disburseFees_vaultBelowMinimum_reverts(
        uint256 _minWithdrawalAmount,
        uint256 _vaultIndex
    )
        public
    {
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

        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        // The entire disbursement should revert because one vault doesn't meet its minimum
        vm.expectRevert("FeeVault: withdrawal amount must be greater than minimum withdrawal amount");
        feeSplitter.disburseFees();

        // Verify no funds were moved (all vaults retain their original balance)
        for (uint256 i = 0; i < 4; i++) {
            uint256 expectedBalance = (i == _vaultIndex) ? insufficientBalance : sufficientBalance;
            assertEq(address(vaults[i]).balance, expectedBalance);
        }
    }

    function _setFeeVaultData(address _vault, uint256 _balance, uint256 _minWithdrawal) internal {
        MockFeeVault mockVault =
            new MockFeeVault(payable(address(feeSplitter)), _minWithdrawal, Types.WithdrawalNetwork.L2);
        vm.deal(address(mockVault), _balance);
        vm.etch(_vault, address(mockVault).code);
        vm.deal(_vault, _balance);
    }

    /// @notice Test that vaults cannot send ETH after disburseFees completes (transient storage cleanup check)
    /// @param _vaultIndex The index of the vault to test (0-3).
    function testFuzz_feeSplitterDisburseFees_vaultsCannotSendAfterDisbursement_reverts(uint256 _vaultIndex) public {
        _vaultIndex = bound(_vaultIndex, 0, 3);

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
        vm.warp(block.timestamp + feeSplitter.feeDisbursementInterval() + 1);

        // Call disburseFees
        feeSplitter.disburseFees();

        // Verify disbursement was successful
        assertEq(feeSplitter.lastDisbursementTime(), block.timestamp);

        // Now try to send ETH from one of the vaults after disbursement
        address _vault = _feeVaults[_vaultIndex];
        uint256 _attemptAmount = 1 ether;
        vm.deal(_vault, _attemptAmount);

        // Attempt to send ETH from the vault - should revert because transient storage was cleared
        vm.prank(_vault);
        vm.expectRevert(IFeeSplitter.FeeSplitter_SenderNotCurrentVault.selector);
        (bool revertsAsExpected,) = payable(address(feeSplitter)).call{ value: _attemptAmount }("");
        assertTrue(revertsAsExpected, "FeeSplitter_Test: call did not revert");
    }
}

/// @title FeeSplitter_SetSharesCalculator_Test
/// @notice Tests the setSharesCalculator function of the `FeeSplitter` contract.
contract FeeSplitter_SetSharesCalculator_Test is FeeSplitter_TestInit {
    /// @notice Test setSharesCalculator reverts when caller is not owner
    function testFuzz_feeSplitterSetSharesCalculator_whenNotOwner_reverts(address _caller) public {
        vm.assume(_caller != _owner);

        vm.prank(_caller);
        vm.expectRevert(IFeeSplitter.FeeSplitter_OnlyProxyAdminOwner.selector);
        feeSplitter.setSharesCalculator(ISharesCalculator(address(0x123)));
    }

    /// @notice Test setSharesCalculator reverts with zero address
    function test_feeSplitterSetSharesCalculator_whenZeroAddress_reverts() public {
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
    function testFuzz_feeSplitterSetFeeDisbursementInterval_whenNotOwner_reverts(address _caller) public {
        vm.assume(_caller != _owner);

        vm.prank(_caller);
        vm.expectRevert(IFeeSplitter.FeeSplitter_OnlyProxyAdminOwner.selector);
        feeSplitter.setFeeDisbursementInterval(48 hours);
    }

    /// @notice Test setFeeDisbursementInterval reverts when interval is zero
    function test_feeSplitterSetFeeDisbursementInterval_whenIntervalZero_reverts() public {
        vm.prank(_owner);
        vm.expectRevert(IFeeSplitter.FeeSplitter_FeeDisbursementIntervalCannotBeZero.selector);
        feeSplitter.setFeeDisbursementInterval(0);
    }

    /// @notice Test setFeeDisbursementInterval reverts when interval is too long
    function testFuzz_feeSplitterSetFeeDisbursementInterval_whenIntervalTooLong_reverts(uint256 _disbursementInterval)
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
