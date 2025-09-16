// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { Reverter } from "test/mocks/Callers.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Contracts
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Types } from "src/libraries/Types.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

/// @title FeeVault_Test
/// @notice Abstract test contract for fee feeVault testing.
///         Subclasses can override the feeVault-specific variables.
abstract contract FeeVault_Test is CommonTest {
    // Variables that can be overridden by concrete test contracts
    address recipient;
    IFeeVault feeVault;
    string feeVaultName;
    uint256 minWithdrawalAmount;
    Types.WithdrawalNetwork expectedWithdrawalNetwork;

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        // Default to L1
        expectedWithdrawalNetwork = Types.WithdrawalNetwork.L1;
        super.setUp();
    }

    /// @notice Helper function to set up L2 withdrawal configuration.
    function _setupL2Withdrawal() internal {
        // Alter the deployment to use WithdrawalNetwork.L2
        vm.etch(
            EIP1967Helper.getImplementation(address(feeVault)),
            address(
                DeployUtils.create1({
                    _name: feeVaultName,
                    _args: DeployUtils.encodeConstructor(
                        abi.encodeCall(
                            IFeeVault.__constructor__,
                            (
                                recipient,
                                minWithdrawalAmount,
                                Types.WithdrawalNetwork.L2
                            )
                        )
                    )
                })
            ).code
        );
    }

    /// @notice Tests that the l1 fee wallet is correct.
    function test_constructor_succeeds() external view virtual {
        assertEq(feeVault.RECIPIENT(), recipient);
        assertEq(feeVault.recipient(), recipient);
        assertEq(feeVault.MIN_WITHDRAWAL_AMOUNT(), minWithdrawalAmount);
        assertEq(feeVault.minWithdrawalAmount(), minWithdrawalAmount);
        assertEq(uint8(feeVault.WITHDRAWAL_NETWORK()), uint8(Types.WithdrawalNetwork.L1));
        assertEq(uint8(feeVault.withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L1));
    }

    /// @notice Tests that the fee feeVault is able to receive ETH.
    function test_receive_succeeds() external {
        uint256 balance = address(feeVault).balance;

        vm.prank(alice);
        (bool success,) = address(feeVault).call{ value: 100 }(hex"");

        assertEq(success, true);
        assertEq(address(feeVault).balance, balance + 100);
    }

    /// @notice Tests that `withdraw` reverts if the balance is less than the minimum withdrawal
    ///         amount.
    function test_withdraw_notEnough_reverts() external {
        assert(address(feeVault).balance < feeVault.MIN_WITHDRAWAL_AMOUNT());

        vm.expectRevert("FeeVault: withdrawal amount must be greater than minimum withdrawal amount");
        feeVault.withdraw();
    }

    /// @notice Tests that `withdraw` successfully initiates a withdrawal to L1.
    function test_withdraw_toL1_succeeds() external {
        uint256 amount = feeVault.MIN_WITHDRAWAL_AMOUNT() + 1;
        vm.deal(address(feeVault), amount);

        // No ether has been withdrawn yet
        assertEq(feeVault.totalProcessed(), 0);

        vm.expectEmit(address(address(feeVault)));
        emit Withdrawal(address(feeVault).balance, recipient, address(this));
        vm.expectEmit(address(address(feeVault)));
        emit Withdrawal(address(feeVault).balance, recipient, address(this), Types.WithdrawalNetwork.L1);

        // The entire feeVault's balance is withdrawn
        vm.expectCall(Predeploys.L2_TO_L1_MESSAGE_PASSER, address(feeVault).balance, hex"");

        // The message is passed to the correct recipient
        vm.expectEmit(Predeploys.L2_TO_L1_MESSAGE_PASSER);
        emit MessagePassed(
            l2ToL1MessagePasser.messageNonce(),
            address(feeVault),
            recipient,
            amount,
            400_000,
            hex"",
            Hashing.hashWithdrawal(
                Types.WithdrawalTransaction({
                    nonce: l2ToL1MessagePasser.messageNonce(),
                    sender: address(feeVault),
                    target: recipient,
                    value: amount,
                    gasLimit: 400_000,
                    data: hex""
                })
            )
        );

        feeVault.withdraw();

        // The withdrawal was successful
        assertEq(feeVault.totalProcessed(), amount);
        assertEq(address(feeVault).balance, 0);
        assertEq(Predeploys.L2_TO_L1_MESSAGE_PASSER.balance, amount);
    }

    /// @notice Tests that `withdraw` successfully initiates a withdrawal to L2.
    function test_withdraw_toL2_succeeds() external {
        _setupL2Withdrawal();

        uint256 amount = feeVault.MIN_WITHDRAWAL_AMOUNT() + 1;
        vm.deal(address(feeVault), amount);

        // No ether has been withdrawn yet
        assertEq(feeVault.totalProcessed(), 0);

        vm.expectEmit(address(address(feeVault)));
        emit Withdrawal(address(feeVault).balance, feeVault.RECIPIENT(), address(this));
        vm.expectEmit(address(address(feeVault)));
        emit Withdrawal(
            address(feeVault).balance, feeVault.RECIPIENT(), address(this), Types.WithdrawalNetwork.L2
        );

        // The entire feeVault's balance is withdrawn
        vm.expectCall(recipient, address(feeVault).balance, bytes(""));

        uint256 withdrawnAmount = feeVault.withdraw();

        // The withdrawal was successful
        assertEq(withdrawnAmount, amount);
        assertEq(feeVault.totalProcessed(), amount);
        assertEq(address(feeVault).balance, 0);
        assertEq(recipient.balance, amount);
    }

    /// @notice Tests that `withdraw` fails if the Recipient reverts. This also serves to simulate
    ///         a situation where insufficient gas is provided to the RECIPIENT.
    function test_withdraw_toL2recipientReverts_fails() external {
        _setupL2Withdrawal();

        uint256 amount = feeVault.MIN_WITHDRAWAL_AMOUNT();

        vm.deal(address(feeVault), amount);
        // No ether has been withdrawn yet
        assertEq(feeVault.totalProcessed(), 0);

        // Ensure the RECIPIENT reverts
        vm.etch(feeVault.RECIPIENT(), type(Reverter).runtimeCode);

        // The entire feeVault's balance is withdrawn
        vm.expectCall(recipient, address(feeVault).balance, bytes(""));
        vm.expectRevert("FeeVault: failed to send ETH to L2 fee recipient");
        feeVault.withdraw();
        assertEq(feeVault.totalProcessed(), 0);
    }

    /// @notice Tests that the owner can successfully set minimum withdrawal amount with fuzz testing.
    function testFuzz_setMinWithdrawalAmount_succeeds(uint256 _newAmount) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        vm.prank(owner);
        IFeeVault(payable(address(feeVault))).setMinWithdrawalAmount(_newAmount);

        // Verify the value was updated
        assertEq(feeVault.minWithdrawalAmount(), _newAmount);
    }

    /// @notice Tests that non-owner cannot set minimum withdrawal amount with fuzz testing.
    function testFuzz_setMinWithdrawalAmount_onlyOwner_reverts(address _caller, uint256 _newAmount) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();
        vm.assume(_caller != owner);

        uint256 initialAmount = feeVault.minWithdrawalAmount();

        vm.prank(_caller);
        vm.expectRevert();
        IFeeVault(payable(address(feeVault))).setMinWithdrawalAmount(_newAmount);

        // Verify the value and boolean flag were NOT changed
        assertEq(feeVault.minWithdrawalAmount(), initialAmount);
    }

    /// @notice Tests that the owner can successfully set recipient with fuzz testing.
    function testFuzz_setRecipient_succeeds(address _newRecipient) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        vm.prank(owner);
        IFeeVault(payable(address(feeVault))).setRecipient(_newRecipient);

        // Verify the value was updated
        assertEq(feeVault.recipient(), _newRecipient);
    }

    /// @notice Tests that non-owner cannot set recipient with fuzz testing.
    function testFuzz_setRecipient_onlyOwner_reverts(address _caller, address _newRecipient) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();
        vm.assume(_caller != owner);

        address initialRecipient = feeVault.recipient();

        vm.prank(_caller);
        vm.expectRevert();
        IFeeVault(payable(address(feeVault))).setRecipient(_newRecipient);

        // Verify the value and boolean flag were NOT changed
        assertEq(feeVault.recipient(), initialRecipient);
    }

    /// @notice Tests that the owner can successfully set withdrawal network with fuzz testing.
    function testFuzz_setWithdrawalNetwork_succeeds(uint8 _networkValue) external {
        // Bound to valid enum values (0 = L1, 1 = L2)
        _networkValue = uint8(bound(_networkValue, 0, 1));
        Types.WithdrawalNetwork newNetwork = Types.WithdrawalNetwork(_networkValue);

        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        vm.prank(owner);
        IFeeVault(payable(address(feeVault))).setWithdrawalNetwork(newNetwork);

        // Verify the value was updated
        assertEq(uint8(feeVault.withdrawalNetwork()), uint8(newNetwork));
    }

    /// @notice Tests that non-owner cannot set withdrawal network with fuzz testing.
    function testFuzz_setWithdrawalNetwork_onlyOwner_reverts(address _caller, uint8 _networkValue) external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();
        vm.assume(_caller != owner);

        // Bound to valid enum values
        _networkValue = uint8(bound(_networkValue, 0, 1));
        Types.WithdrawalNetwork newNetwork = Types.WithdrawalNetwork(_networkValue);

        Types.WithdrawalNetwork initialNetwork = feeVault.withdrawalNetwork();

        vm.prank(_caller);
        vm.expectRevert();
        IFeeVault(payable(address(feeVault))).setWithdrawalNetwork(newNetwork);

        // Verify the value and boolean flag were NOT changed
        assertEq(uint8(feeVault.withdrawalNetwork()), uint8(initialNetwork));
    }

    /// @notice Tests that minWithdrawalAmount returns immutable by default, then storage after being set.
    function test_minWithdrawalAmount_returnsImmutableThenStorage_succeeds() external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        // Initially should return the immutable value
        uint256 immutableValue = feeVault.MIN_WITHDRAWAL_AMOUNT();
        assertEq(feeVault.minWithdrawalAmount(), immutableValue);

        // Set a different value via owner
        uint256 newValue = immutableValue + 1 ether;
        vm.prank(owner);
        IFeeVault(payable(address(feeVault))).setMinWithdrawalAmount(newValue);

        // Now should return the storage value, not the immutable
        assertEq(feeVault.minWithdrawalAmount(), newValue);
        assertNotEq(feeVault.minWithdrawalAmount(), immutableValue);
        assertEq(feeVault.MIN_WITHDRAWAL_AMOUNT(), immutableValue); // immutable unchanged
    }

    /// @notice Tests that recipient returns immutable by default, then storage after being set.
    function test_recipient_returnsImmutableThenStorage_succeeds() external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        // Initially should return the immutable value
        address immutableValue = feeVault.RECIPIENT();
        assertEq(feeVault.recipient(), immutableValue);

        // Set a different value via owner
        address newValue = address(0x123);
        vm.prank(owner);
        IFeeVault(payable(address(feeVault))).setRecipient(newValue);

        // Now should return the storage value, not the immutable
        assertEq(feeVault.recipient(), newValue);
        assertNotEq(feeVault.recipient(), immutableValue);
        assertEq(feeVault.RECIPIENT(), immutableValue); // immutable unchanged
    }

    /// @notice Tests that withdrawalNetwork returns immutable by default, then storage after being set.
    function test_withdrawalNetwork_returnsImmutableThenStorage_succeeds() external {
        address owner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        // Initially should return the immutable value
        Types.WithdrawalNetwork immutableValue = feeVault.WITHDRAWAL_NETWORK();
        assertEq(uint8(feeVault.withdrawalNetwork()), uint8(immutableValue));

        // Set a different value via owner (toggle between L1 and L2)
        Types.WithdrawalNetwork newValue = immutableValue == Types.WithdrawalNetwork.L1
            ? Types.WithdrawalNetwork.L2
            : Types.WithdrawalNetwork.L1;
        vm.prank(owner);
        IFeeVault(payable(address(feeVault))).setWithdrawalNetwork(newValue);

        // Now should return the storage value, not the immutable
        assertEq(uint8(feeVault.withdrawalNetwork()), uint8(newValue));
        assertNotEq(uint8(feeVault.withdrawalNetwork()), uint8(immutableValue));
        assertEq(uint8(feeVault.WITHDRAWAL_NETWORK()), uint8(immutableValue)); // immutable unchanged
    }
}