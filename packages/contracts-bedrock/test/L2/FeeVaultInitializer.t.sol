// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Contracts
import { FeeVaultInitializer } from "src/L2/FeeVaultInitializer.sol";

// Mocks
import { MockLegacyFeeVault } from "test/mocks/MockFeeVault.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IBaseFeeVault } from "interfaces/L2/IBaseFeeVault.sol";
import { ISequencerFeeVault } from "interfaces/L2/ISequencerFeeVault.sol";
import { IL1FeeVault } from "interfaces/L2/IL1FeeVault.sol";
import { IOperatorFeeVault } from "interfaces/L2/IOperatorFeeVault.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

/// @title FeeVaultInitializer_Constructor_Test
/// @notice Test contract for the FeeVaultInitializer contract's functionality
contract FeeVaultInitializer_Constructor_Test is CommonTest {
    FeeVaultInitializer feeVaultInitializer;

    // Store original vault configurations
    address originalBaseRecipient;
    uint256 originalBaseMinWithdrawal;
    Types.WithdrawalNetwork originalBaseNetwork;

    address originalSequencerRecipient;
    uint256 originalSequencerMinWithdrawal;
    Types.WithdrawalNetwork originalSequencerNetwork;

    address originalL1Recipient;
    uint256 originalL1MinWithdrawal;
    Types.WithdrawalNetwork originalL1Network;

    address originalOperatorRecipient;
    uint256 originalOperatorMinWithdrawal;
    Types.WithdrawalNetwork originalOperatorNetwork;

    event FeeVaultDeployed(
        string indexed vaultType,
        address indexed newImplementation,
        address recipient,
        Types.WithdrawalNetwork network,
        uint256 minWithdrawalAmount
    );

    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }

    function setUp() public override {
        super.setUp();

        // Capture original Base Fee Vault configuration
        originalBaseRecipient = baseFeeVault.RECIPIENT();
        originalBaseMinWithdrawal = baseFeeVault.MIN_WITHDRAWAL_AMOUNT();
        originalBaseNetwork = baseFeeVault.WITHDRAWAL_NETWORK();

        // Capture original Sequencer Fee Vault configuration
        originalSequencerRecipient = sequencerFeeVault.RECIPIENT();
        originalSequencerMinWithdrawal = sequencerFeeVault.MIN_WITHDRAWAL_AMOUNT();
        originalSequencerNetwork = sequencerFeeVault.WITHDRAWAL_NETWORK();

        // Capture original L1 Fee Vault configuration
        originalL1Recipient = l1FeeVault.RECIPIENT();
        originalL1MinWithdrawal = l1FeeVault.MIN_WITHDRAWAL_AMOUNT();
        originalL1Network = l1FeeVault.WITHDRAWAL_NETWORK();

        // Capture original Operator Fee Vault configuration
        originalOperatorRecipient = operatorFeeVault.RECIPIENT();
        originalOperatorMinWithdrawal = operatorFeeVault.MIN_WITHDRAWAL_AMOUNT();
        originalOperatorNetwork = operatorFeeVault.WITHDRAWAL_NETWORK();
    }

    function test_constructor_succeeds() public {
        // Get the current nonce and predicted initializer address to predict the vault addresses
        uint64 currentNonce = vm.getNonce(address(this));
        address predictedInitializerAddress = vm.computeCreateAddress(address(this), currentNonce);

        // Test event emissions before fee vault initializer deployment
        _testEventEmissions(predictedInitializerAddress);

        // Deploy the FeeVaultInitializer
        feeVaultInitializer = new FeeVaultInitializer();

        // Can now read the fee vault initializer version
        assertEq(feeVaultInitializer.version(), "1.0.0");
        assertEq(address(feeVaultInitializer), predictedInitializerAddress);

        // Test the new implementations have correct configurations against the original values
        _testNewImplementations(predictedInitializerAddress);
    }

    function test_constructor_withLegacyBaseFeeVault_succeeds() public {
        // Deploy the legacy mock vault
        MockLegacyFeeVault legacyVault = new MockLegacyFeeVault();

        // Use vm.etch to replace the base fee vault predeploy with our legacy mock
        // This simulates an old vault that doesn't have the WITHDRAWAL_NETWORK function
        vm.etch(Predeploys.BASE_FEE_VAULT, address(legacyVault).code);

        // Get the current nonce and predicted initializer address
        uint64 currentNonce = vm.getNonce(address(this));
        address predictedInitializerAddress = vm.computeCreateAddress(address(this), currentNonce);
        address predictedBaseFeeVault = vm.computeCreateAddress(predictedInitializerAddress, 1);

        // Expect the FeeVaultDeployed event with default L2 network for the legacy vault
        vm.expectEmit(predictedInitializerAddress);
        emit FeeVaultDeployed(
            "BaseFeeVault",
            predictedBaseFeeVault,
            legacyVault.RECIPIENT(),
            Types.WithdrawalNetwork.L2, // Should default to L2
            legacyVault.MIN_WITHDRAWAL_AMOUNT()
        );

        // Deploy the FeeVaultInitializer - this should handle the legacy vault gracefully
        feeVaultInitializer = new FeeVaultInitializer();

        // Verify the implementation was deployed correctly with default L2 network
        IBaseFeeVault newBaseFeeVault = IBaseFeeVault(payable(predictedBaseFeeVault));
        assertEq(newBaseFeeVault.RECIPIENT(), legacyVault.RECIPIENT());
        assertEq(newBaseFeeVault.MIN_WITHDRAWAL_AMOUNT(), legacyVault.MIN_WITHDRAWAL_AMOUNT());
        assertEq(uint8(newBaseFeeVault.WITHDRAWAL_NETWORK()), uint8(Types.WithdrawalNetwork.L2));

        // Check new getter functions also return the correct values
        assertEq(newBaseFeeVault.recipient(), legacyVault.RECIPIENT());
        assertEq(newBaseFeeVault.minWithdrawalAmount(), legacyVault.MIN_WITHDRAWAL_AMOUNT());
        assertEq(uint8(newBaseFeeVault.withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L2));
    }

    function test_constructor_whenVaultsWithdrawalNetworkIsL2_succeeds() public {
        // Mock the calls to the fee vaults to return L2 as the withdrawal network
        _mockAndExpect(
            Predeploys.BASE_FEE_VAULT,
            abi.encodeCall(IBaseFeeVault.WITHDRAWAL_NETWORK, ()),
            abi.encode(Types.WithdrawalNetwork.L2)
        );
        _mockAndExpect(
            Predeploys.SEQUENCER_FEE_WALLET,
            abi.encodeCall(ISequencerFeeVault.WITHDRAWAL_NETWORK, ()),
            abi.encode(Types.WithdrawalNetwork.L2)
        );
        _mockAndExpect(
            Predeploys.L1_FEE_VAULT,
            abi.encodeCall(IL1FeeVault.WITHDRAWAL_NETWORK, ()),
            abi.encode(Types.WithdrawalNetwork.L2)
        );
        _mockAndExpect(
            Predeploys.OPERATOR_FEE_VAULT,
            abi.encodeCall(IOperatorFeeVault.WITHDRAWAL_NETWORK, ()),
            abi.encode(Types.WithdrawalNetwork.L2)
        );

        address predictedInitializerAddress = vm.computeCreateAddress(address(this), vm.getNonce(address(this)));
        address predictedBaseFeeVault = vm.computeCreateAddress(predictedInitializerAddress, 1);
        address predictedSequencerFeeVault = vm.computeCreateAddress(predictedInitializerAddress, 2);
        address predictedL1FeeVault = vm.computeCreateAddress(predictedInitializerAddress, 3);
        address predictedOperatorFeeVault = vm.computeCreateAddress(predictedInitializerAddress, 4);

        // Deploy the FeeVaultInitializer
        feeVaultInitializer = new FeeVaultInitializer();

        // Check the vault's withdrawal network is L2
        assertEq(
            uint8(IFeeVault(payable(predictedBaseFeeVault)).withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L2)
        );
        assertEq(
            uint8(IFeeVault(payable(predictedSequencerFeeVault)).withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L2)
        );
        assertEq(uint8(IFeeVault(payable(predictedL1FeeVault)).withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L2));
        assertEq(
            uint8(IFeeVault(payable(predictedOperatorFeeVault)).withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L2)
        );
    }

    function test_constructor_withLegacySequencerFeeVault_succeeds() public {
        // Deploy the legacy mock vault
        MockLegacyFeeVault legacyVault = new MockLegacyFeeVault();

        // Use vm.etch to replace the sequencer fee wallet predeploy with our legacy mock
        // This simulates an old vault that doesn't have the WITHDRAWAL_NETWORK function
        vm.etch(Predeploys.SEQUENCER_FEE_WALLET, address(legacyVault).code);

        // Get the current nonce and predicted initializer address
        uint64 currentNonce = vm.getNonce(address(this));
        address predictedInitializerAddress = vm.computeCreateAddress(address(this), currentNonce);
        address predictedSequencerFeeVault = vm.computeCreateAddress(predictedInitializerAddress, 2);

        // Expect the FeeVaultDeployed event with default L2 network for the legacy vault
        vm.expectEmit(predictedInitializerAddress);
        emit FeeVaultDeployed(
            "SequencerFeeVault",
            predictedSequencerFeeVault,
            legacyVault.RECIPIENT(),
            Types.WithdrawalNetwork.L2, // Should default to L2
            legacyVault.MIN_WITHDRAWAL_AMOUNT()
        );

        // Deploy the FeeVaultInitializer - this should handle the legacy vault gracefully
        feeVaultInitializer = new FeeVaultInitializer();

        // Verify the implementation was deployed correctly with default L2 network
        ISequencerFeeVault newSequencerFeeVault = ISequencerFeeVault(payable(predictedSequencerFeeVault));
        assertEq(newSequencerFeeVault.RECIPIENT(), legacyVault.RECIPIENT());
        assertEq(newSequencerFeeVault.MIN_WITHDRAWAL_AMOUNT(), legacyVault.MIN_WITHDRAWAL_AMOUNT());
        assertEq(uint8(newSequencerFeeVault.WITHDRAWAL_NETWORK()), uint8(Types.WithdrawalNetwork.L2));

        // Check new getter functions also return the correct values
        assertEq(newSequencerFeeVault.recipient(), legacyVault.RECIPIENT());
        assertEq(newSequencerFeeVault.minWithdrawalAmount(), legacyVault.MIN_WITHDRAWAL_AMOUNT());
        assertEq(uint8(newSequencerFeeVault.withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L2));
    }

    function test_constructor_withLegacyL1FeeVault_succeeds() public {
        // Deploy the legacy mock vault
        MockLegacyFeeVault legacyVault = new MockLegacyFeeVault();

        // Use vm.etch to replace the l1 fee vault predeploy with our legacy mock
        // This simulates an old vault that doesn't have the WITHDRAWAL_NETWORK function
        vm.etch(Predeploys.L1_FEE_VAULT, address(legacyVault).code);

        // Get the current nonce and predicted initializer address
        uint64 currentNonce = vm.getNonce(address(this));
        address predictedInitializerAddress = vm.computeCreateAddress(address(this), currentNonce);
        address predictedL1FeeVault = vm.computeCreateAddress(predictedInitializerAddress, 3);

        // Expect the FeeVaultDeployed event with default L2 network for the legacy vault
        vm.expectEmit(predictedInitializerAddress);
        emit FeeVaultDeployed(
            "L1FeeVault",
            predictedL1FeeVault,
            legacyVault.RECIPIENT(),
            Types.WithdrawalNetwork.L2, // Should default to L2
            legacyVault.MIN_WITHDRAWAL_AMOUNT()
        );

        // Deploy the FeeVaultInitializer - this should handle the legacy vault gracefully
        feeVaultInitializer = new FeeVaultInitializer();

        // Verify the implementation was deployed correctly with default L2 network
        IL1FeeVault newL1FeeVault = IL1FeeVault(payable(predictedL1FeeVault));
        assertEq(newL1FeeVault.RECIPIENT(), legacyVault.RECIPIENT());
        assertEq(newL1FeeVault.MIN_WITHDRAWAL_AMOUNT(), legacyVault.MIN_WITHDRAWAL_AMOUNT());
        assertEq(uint8(newL1FeeVault.WITHDRAWAL_NETWORK()), uint8(Types.WithdrawalNetwork.L2));

        // Check new getter functions also return the correct values
        assertEq(newL1FeeVault.recipient(), legacyVault.RECIPIENT());
        assertEq(newL1FeeVault.minWithdrawalAmount(), legacyVault.MIN_WITHDRAWAL_AMOUNT());
        assertEq(uint8(newL1FeeVault.withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L2));
    }

    function test_constructor_withLegacyOperatorFeeVault_succeeds() public {
        // Deploy the legacy mock vault
        MockLegacyFeeVault legacyVault = new MockLegacyFeeVault();

        // Use vm.etch to replace the operator fee vault predeploy with our legacy mock
        // This simulates an old vault that doesn't have the WITHDRAWAL_NETWORK function
        vm.etch(Predeploys.OPERATOR_FEE_VAULT, address(legacyVault).code);

        // Get the current nonce and predicted initializer address
        uint64 currentNonce = vm.getNonce(address(this));
        address predictedInitializerAddress = vm.computeCreateAddress(address(this), currentNonce);
        address predictedOperatorFeeVault = vm.computeCreateAddress(predictedInitializerAddress, 4);

        // Expect the FeeVaultDeployed event with default L2 network for the legacy vault
        vm.expectEmit(predictedInitializerAddress);
        emit FeeVaultDeployed(
            "OperatorFeeVault",
            predictedOperatorFeeVault,
            legacyVault.RECIPIENT(),
            Types.WithdrawalNetwork.L2, // Should default to L2
            legacyVault.MIN_WITHDRAWAL_AMOUNT()
        );

        // Deploy the FeeVaultInitializer - this should handle the legacy vault gracefully
        feeVaultInitializer = new FeeVaultInitializer();

        // Verify the implementation was deployed correctly with default L2 network
        IOperatorFeeVault newOperatorFeeVault = IOperatorFeeVault(payable(predictedOperatorFeeVault));
        assertEq(newOperatorFeeVault.RECIPIENT(), legacyVault.RECIPIENT());
        assertEq(newOperatorFeeVault.MIN_WITHDRAWAL_AMOUNT(), legacyVault.MIN_WITHDRAWAL_AMOUNT());
        assertEq(uint8(newOperatorFeeVault.WITHDRAWAL_NETWORK()), uint8(Types.WithdrawalNetwork.L2));

        // Check new getter functions also return the correct values
        assertEq(newOperatorFeeVault.recipient(), legacyVault.RECIPIENT());
        assertEq(newOperatorFeeVault.minWithdrawalAmount(), legacyVault.MIN_WITHDRAWAL_AMOUNT());
        assertEq(uint8(newOperatorFeeVault.withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L2));
    }

    function _testEventEmissions(address _predictedInitializerAddress) internal {
        address predictedBaseFeeVault = vm.computeCreateAddress(_predictedInitializerAddress, 1);
        address predictedSequencerFeeVault = vm.computeCreateAddress(_predictedInitializerAddress, 2);
        address predictedL1FeeVault = vm.computeCreateAddress(_predictedInitializerAddress, 3);
        address predictedOperatorFeeVault = vm.computeCreateAddress(_predictedInitializerAddress, 4);

        // Expect the FeeVaultDeployed events from the FeeVaultInitializer contract using the original values
        vm.expectEmit(_predictedInitializerAddress);
        emit FeeVaultDeployed(
            "BaseFeeVault", predictedBaseFeeVault, originalBaseRecipient, originalBaseNetwork, originalBaseMinWithdrawal
        );

        vm.expectEmit(_predictedInitializerAddress);
        emit FeeVaultDeployed(
            "SequencerFeeVault",
            predictedSequencerFeeVault,
            originalSequencerRecipient,
            originalSequencerNetwork,
            originalSequencerMinWithdrawal
        );

        vm.expectEmit(_predictedInitializerAddress);
        emit FeeVaultDeployed(
            "L1FeeVault", predictedL1FeeVault, originalL1Recipient, originalL1Network, originalL1MinWithdrawal
        );

        vm.expectEmit(_predictedInitializerAddress);
        emit FeeVaultDeployed(
            "OperatorFeeVault",
            predictedOperatorFeeVault,
            originalOperatorRecipient,
            originalOperatorNetwork,
            originalOperatorMinWithdrawal
        );
    }

    function _testNewImplementations(address _predictedInitializerAddress) internal view {
        address predictedBaseFeeVault = vm.computeCreateAddress(_predictedInitializerAddress, 1);
        address predictedSequencerFeeVault = vm.computeCreateAddress(_predictedInitializerAddress, 2);
        address predictedL1FeeVault = vm.computeCreateAddress(_predictedInitializerAddress, 3);
        address predictedOperatorFeeVault = vm.computeCreateAddress(_predictedInitializerAddress, 4);

        _testBaseFeeVaultImplementation(predictedBaseFeeVault);
        _testSequencerFeeVaultImplementation(predictedSequencerFeeVault);
        _testL1FeeVaultImplementation(predictedL1FeeVault);
        _testOperatorFeeVaultImplementation(predictedOperatorFeeVault);
    }

    function _testBaseFeeVaultImplementation(address _newImplementation) internal view {
        IBaseFeeVault newBaseFeeVault = IBaseFeeVault(payable(_newImplementation));
        // Test against the original stored values
        assertEq(newBaseFeeVault.RECIPIENT(), originalBaseRecipient);
        assertEq(newBaseFeeVault.MIN_WITHDRAWAL_AMOUNT(), originalBaseMinWithdrawal);
        assertEq(uint8(newBaseFeeVault.WITHDRAWAL_NETWORK()), uint8(originalBaseNetwork));

        // Check new getter functions return the same original values
        assertEq(newBaseFeeVault.recipient(), originalBaseRecipient);
        assertEq(newBaseFeeVault.minWithdrawalAmount(), originalBaseMinWithdrawal);
        assertEq(uint8(newBaseFeeVault.withdrawalNetwork()), uint8(originalBaseNetwork));
    }

    function _testSequencerFeeVaultImplementation(address _newImplementation) internal view {
        ISequencerFeeVault newSequencerFeeVault = ISequencerFeeVault(payable(_newImplementation));
        // Test against the original stored values
        assertEq(newSequencerFeeVault.RECIPIENT(), originalSequencerRecipient);
        assertEq(newSequencerFeeVault.MIN_WITHDRAWAL_AMOUNT(), originalSequencerMinWithdrawal);
        assertEq(uint8(newSequencerFeeVault.WITHDRAWAL_NETWORK()), uint8(originalSequencerNetwork));

        // Check new getter functions return the same original values
        assertEq(newSequencerFeeVault.recipient(), originalSequencerRecipient);
        assertEq(newSequencerFeeVault.minWithdrawalAmount(), originalSequencerMinWithdrawal);
        assertEq(uint8(newSequencerFeeVault.withdrawalNetwork()), uint8(originalSequencerNetwork));
    }

    function _testL1FeeVaultImplementation(address _newImplementation) internal view {
        IL1FeeVault newL1FeeVault = IL1FeeVault(payable(_newImplementation));
        // Test against the original stored values
        assertEq(newL1FeeVault.RECIPIENT(), originalL1Recipient);
        assertEq(newL1FeeVault.MIN_WITHDRAWAL_AMOUNT(), originalL1MinWithdrawal);
        assertEq(uint8(newL1FeeVault.WITHDRAWAL_NETWORK()), uint8(originalL1Network));

        // Check new getter functions return the same original values
        assertEq(newL1FeeVault.recipient(), originalL1Recipient);
        assertEq(newL1FeeVault.minWithdrawalAmount(), originalL1MinWithdrawal);
        assertEq(uint8(newL1FeeVault.withdrawalNetwork()), uint8(originalL1Network));
    }

    function _testOperatorFeeVaultImplementation(address _newImplementation) internal view {
        IOperatorFeeVault newOperatorFeeVault = IOperatorFeeVault(payable(_newImplementation));
        // Test against the original stored values
        assertEq(newOperatorFeeVault.RECIPIENT(), originalOperatorRecipient);
        assertEq(newOperatorFeeVault.MIN_WITHDRAWAL_AMOUNT(), originalOperatorMinWithdrawal);
        assertEq(uint8(newOperatorFeeVault.WITHDRAWAL_NETWORK()), uint8(originalOperatorNetwork));

        // Check new getter functions return the same original values
        assertEq(newOperatorFeeVault.recipient(), originalOperatorRecipient);
        assertEq(newOperatorFeeVault.minWithdrawalAmount(), originalOperatorMinWithdrawal);
        assertEq(uint8(newOperatorFeeVault.withdrawalNetwork()), uint8(originalOperatorNetwork));
    }
}
