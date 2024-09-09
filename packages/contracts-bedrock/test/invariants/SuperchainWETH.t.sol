// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { StdUtils } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";

import { Predeploys } from "src/libraries/Predeploys.sol";
import { SuperchainWETH } from "src/L2/SuperchainWETH.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/interfaces/IL2ToL2CrossDomainMessenger.sol";

import { CommonTest } from "test/setup/CommonTest.sol";

/// @title SuperchainWETH_User
/// @notice Actor contract that interacts with the SuperchainWETH contract.
contract SuperchainWETH_User is StdUtils {
    /// @notice Cross domain message data.
    struct MessageData {
        bytes32 id;
        uint256 amount;
    }

    /// @notice Flag to indicate if the test has failed.
    bool public failed = false;

    /// @notice The Vm contract.
    Vm internal vm;

    /// @notice The SuperchainWETH contract.
    SuperchainWETH internal weth;

    /// @notice Mapping of sent messages.
    mapping(bytes32 => bool) internal sent;

    /// @notice Array of unrelayed messages.
    MessageData[] internal unrelayed;

    /// @param _vm The Vm contract.
    /// @param _weth The SuperchainWETH contract.
    /// @param _balance The initial balance of the contract.
    constructor(Vm _vm, SuperchainWETH _weth, uint256 _balance) {
        vm = _vm;
        weth = _weth;
        vm.deal(address(this), _balance);
    }

    /// @notice Allow the contract to receive ETH.
    receive() external payable { }

    /// @notice Deposit ETH into the contract.
    /// @param _amount The amount of ETH to deposit.
    function deposit(uint256 _amount) public {
        // Bound deposit amount to our ETH balance.
        _amount = bound(_amount, 0, address(this).balance);

        // Deposit the amount.
        try weth.deposit{ value: _amount }() {
            // Success.
        } catch {
            failed = true;
        }
    }

    /// @notice Withdraw ETH from the contract.
    /// @param _amount The amount of ETH to withdraw.
    function withdraw(uint256 _amount) public {
        // Bound withdraw amount to our WETH balance.
        _amount = bound(_amount, 0, weth.balanceOf(address(this)));

        // Withdraw the amount.
        try weth.withdraw(_amount) {
            // Success.
        } catch {
            failed = true;
        }
    }

    /// @notice Send ERC20 tokens to another chain.
    /// @param _amount The amount of ERC20 tokens to send.
    /// @param _chainId The chain ID to send the tokens to.
    /// @param _messageId The message ID.
    function sendERC20(uint256 _amount, uint256 _chainId, bytes32 _messageId) public {
        // Make sure we aren't reusing a message ID.
        if (sent[_messageId]) {
            return;
        }

        // Bound send amount to our WETH balance.
        _amount = bound(_amount, 0, weth.balanceOf(address(this)));

        // Prevent receiving chain ID from being the same as the current chain ID.
        _chainId = _chainId == block.chainid ? _chainId + 1 : _chainId;

        // Send the amount.
        try weth.sendERC20(address(this), _amount, _chainId) {
            // Success.
        } catch {
            failed = true;
        }

        // Mark message as sent.
        sent[_messageId] = true;
        unrelayed.push(MessageData({ id: _messageId, amount: _amount }));
    }

    /// @notice Relay a message from another chain.
    function relayMessage(uint256 _source) public {
        // Make sure there are unrelayed messages.
        if (unrelayed.length == 0) {
            return;
        }

        // Grab the latest unrelayed message.
        MessageData memory message = unrelayed[unrelayed.length - 1];

        // Simulate the cross-domain message.
        // Make sure the cross-domain message sender is set to this contract.
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageSender, ()),
            abi.encode(address(weth))
        );

        // Simulate the cross-domain message source to any chain.
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageSource, ()),
            abi.encode(_source)
        );

        // Prank the relayERC20 function.
        // Balance will just go back to our own account.
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        try weth.relayERC20(address(this), address(this), message.amount) {
            // Success.
        } catch {
            failed = true;
        }

        // Remove the message from the unrelayed list.
        unrelayed.pop();
    }
}

/// @title SuperchainWETH_SendSucceeds_Invariant
/// @notice Invariant test that checks that sending WETH always succeeds if the actor has a
///         sufficient balance to do so and that the actor's balance does not increase out of
///         nowhere.
contract SuperchainWETH_SendSucceeds_Invariant is CommonTest {
    /// @notice Starting balance of the contract.
    uint256 internal constant STARTING_BALANCE = type(uint248).max;

    /// @notice The SuperchainWETH_User actor.
    SuperchainWETH_User internal actor;

    /// @notice Test setup.
    function setUp() public override {
        super.enableInterop();
        super.setUp();

        // Create a new SuperchainWETH_User actor.
        actor = new SuperchainWETH_User(vm, superchainWeth, STARTING_BALANCE);

        // Set the target contract.
        targetContract(address(actor));

        // Set the target selectors.
        bytes4[] memory selectors = new bytes4[](4);
        selectors[0] = actor.deposit.selector;
        selectors[1] = actor.withdraw.selector;
        selectors[2] = actor.sendERC20.selector;
        selectors[3] = actor.relayMessage.selector;
        FuzzSelector memory selector = FuzzSelector({ addr: address(actor), selectors: selectors });
        targetSelector(selector);
    }

    /// @notice Invariant that checks that sending WETH always succeeds.
    /// @custom:invariant Calls to sendERC20 should always succeed as long as the actor has less
    ///                   than uint248 wei which is much greater than the total ETH supply. Actor's
    ///                   balance should also not increase out of nowhere.
    function invariant_sendERC20_succeeds() public view {
        // Assert that the actor has not failed to send WETH.
        assertEq(actor.failed(), false);

        // Assert that the actor's balance has not somehow increased.
        assertLe(address(actor).balance, STARTING_BALANCE);
    }
}
