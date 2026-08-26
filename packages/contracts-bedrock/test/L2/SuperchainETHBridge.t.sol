// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { MockHelper } from "test/utils/MockHelper.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Unauthorized, ZeroAddress } from "src/libraries/errors/CommonErrors.sol";

// Interfaces
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";
import { ISuperchainETHBridge } from "interfaces/L2/ISuperchainETHBridge.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @title SuperchainETHBridge_TestInit
/// @notice Reusable test initialization for `SuperchainETHBridge` tests.
abstract contract SuperchainETHBridge_TestInit is CommonTest, MockHelper {
    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);

    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);

    address internal constant ZERO_ADDRESS = address(0);

    /// @notice Fixed counterpart chain IDs used by the net-flow cap cases.
    uint256 internal constant CHAIN_P = 424_243;
    uint256 internal constant CHAIN_B = 8453;

    /// @notice Test setup.
    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();

        {
            // TODO: Remove this block when L2Genesis includes this contract.
            vm.etch(address(superchainETHBridge), vm.getDeployedCode("SuperchainETHBridge.sol:SuperchainETHBridge"));
            vm.etch(address(ethLiquidity), vm.getDeployedCode("ETHLiquidity.sol:ETHLiquidity"));
        }
    }

    /// @notice Credits the net-flow cap for `_chainId` the only way it can ever be credited: by
    ///         actually sending ETH to that chain. Clears mocked calls on the way out so callers can
    ///         install their own `crossDomainMessageContext` mock afterwards.
    function _sendToChain(uint256 _chainId, uint256 _amount) internal {
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeWithSelector(IL2ToL2CrossDomainMessenger.sendMessage.selector),
            abi.encode(bytes32(0))
        );

        address _funder = address(uint160(uint256(keccak256("netflow.funder"))));
        vm.deal(_funder, _amount);
        vm.prank(_funder);
        superchainETHBridge.sendETH{ value: _amount }(address(uint160(uint256(keccak256("netflow.to")))), _chainId);

        vm.clearMockedCalls();
    }

    /// @notice Relays `_amount` as if it came from `_source`, with the cross domain context mocked.
    function _relayFrom(uint256 _source, address _from, address _to, uint256 _amount) internal {
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(address(superchainETHBridge), _source)
        );
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        superchainETHBridge.relayETH(_from, _to, _amount);
    }
}

/// @title SuperchainETHBridge_SendETH_Test
/// @notice Tests the `sendETH` function of the `SuperchainETHBridge` contract.
contract SuperchainETHBridge_SendETH_Test is SuperchainETHBridge_TestInit {
    /// @notice Tests the `sendETH` function reverts when the address `_to` is zero.
    function testFuzz_sendETH_zeroAddressTo_reverts(address _sender, uint256 _amount, uint256 _chainId) public {
        // Expect the revert with `ZeroAddress` selector
        vm.expectRevert(ZeroAddress.selector);

        vm.deal(_sender, _amount);
        vm.prank(_sender);
        // Call the `sendETH` function with the zero address as `_to`
        superchainETHBridge.sendETH{ value: _amount }(ZERO_ADDRESS, _chainId);
    }

    /// @notice Tests the `sendETH` function burns the sender ETH, sends the message, and emits the
    ///         `SendETH` event.
    function testFuzz_sendETH_succeeds(
        address _sender,
        address _to,
        uint256 _amount,
        uint256 _chainId,
        bytes32 _msgHash
    )
        external
    {
        // Assume
        vm.assume(_sender != address(ethLiquidity));
        vm.assume(_sender != ZERO_ADDRESS);
        vm.assume(_to != ZERO_ADDRESS);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(_sender, _amount);

        // Get the total balance of `_sender` before the send to compare later on the assertions
        uint256 _senderBalanceBefore = _sender.balance;
        uint256 _netSentBefore = superchainETHBridge.netSent(_chainId);

        // Look for the emit of the `SendETH` event
        vm.expectEmit(address(superchainETHBridge));
        emit SendETH(_sender, _to, _amount, _chainId);

        // Expect the call to the `burn` function in the `ETHLiquidity` contract
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.burn, ()), 1);

        // Mock the call over the `sendMessage` function and expect it to be called properly
        bytes memory _message = abi.encodeCall(superchainETHBridge.relayETH, (_sender, _to, _amount));
        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.sendMessage, (_chainId, address(superchainETHBridge), _message)),
            abi.encode(_msgHash)
        );

        // Call the `sendETH` function
        vm.prank(_sender);
        bytes32 _returnedMsgHash = superchainETHBridge.sendETH{ value: _amount }(_to, _chainId);

        // Check the message hash was generated correctly
        assertEq(_msgHash, _returnedMsgHash);

        // Check the total supply and balance of `_sender` after the send were updated correctly
        assertEq(_sender.balance, _senderBalanceBefore - _amount);

        // Check the destination chain was credited exactly what was sent to it
        assertEq(superchainETHBridge.netSent(_chainId), _netSentBefore + _amount);
    }

    /// @notice A zero-value send credits nothing, so it grants no extraction allowance.
    function testFuzz_sendETH_zeroValue_succeeds(uint256 _chainId) public {
        _sendToChain(_chainId, 0);
        assertEq(superchainETHBridge.netSent(_chainId), 0);

        vm.expectRevert(ISuperchainETHBridge.InsufficientNetFlow.selector);
        _relayFrom(_chainId, address(this), address(0xBEEF), 1 wei);
    }

    /// @notice Repeated sends accumulate into one cap for the destination chain.
    function testFuzz_sendETH_repeatedSends_succeeds(uint256 _chainId, uint96 _first, uint96 _second) public {
        _sendToChain(_chainId, _first);
        assertEq(superchainETHBridge.netSent(_chainId), uint256(_first));

        _sendToChain(_chainId, _second);
        assertEq(superchainETHBridge.netSent(_chainId), uint256(_first) + uint256(_second));
    }
}

/// @title SuperchainETHBridge_RelayETH_Test
/// @notice Tests the `relayETH` function of the `SuperchainETHBridge` contract.
contract SuperchainETHBridge_RelayETH_Test is SuperchainETHBridge_TestInit {
    /// @notice Tests the `relayETH` function reverts when the caller is not the
    ///         `L2ToL2CrossDomainMessenger`.
    function testFuzz_relayETH_notMessenger_reverts(address _caller, address _to, uint256 _amount) public {
        // Ensure the caller is not the messenger
        vm.assume(_caller != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(Unauthorized.selector);

        // Call the `relayETH` function with the non-messenger caller
        vm.prank(_caller);
        superchainETHBridge.relayETH(_caller, _to, _amount);
    }

    /// @notice Tests the `relayETH` function reverts when the `crossDomainMessageSender` that sent
    ///         the message is not the same `SuperchainETHBridge`.
    function testFuzz_relayETH_notCrossDomainSender_reverts(
        address _crossDomainMessageSender,
        uint256 _source,
        address _to,
        uint256 _amount
    )
        public
    {
        vm.assume(_crossDomainMessageSender != address(superchainETHBridge));

        // Mock the call over the `crossDomainMessageContext` function setting a wrong sender
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(_crossDomainMessageSender, _source)
        );

        // Expect the revert with `InvalidCrossDomainSender` selector
        vm.expectRevert(ISuperchainETHBridge.InvalidCrossDomainSender.selector);

        // Call the `relayETH` function with the sender caller
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        superchainETHBridge.relayETH(_crossDomainMessageSender, _to, _amount);
    }

    /// @notice Tests the `relayETH` function relays the proper amount of ETH and emits the
    ///         `RelayETH` event.
    function testFuzz_relayETH_succeeds(address _from, address _to, uint256 _amount, uint256 _source) public {
        // Assume
        vm.assume(_to != ZERO_ADDRESS);
        assumePayable(_to);
        _amount = bound(_amount, 0, type(uint248).max - 1);

        // Arrange
        vm.deal(address(superchainETHBridge), _amount);
        vm.deal(Predeploys.ETH_LIQUIDITY, _amount);

        // Credit the net-flow cap for the source chain: relays are only allowed up to what this
        // chain has actually sent there.
        _sendToChain(_source, _amount);

        _mockAndExpect(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(address(superchainETHBridge), _source)
        );

        uint256 _toBalanceBefore = _to.balance;

        // Look for the emit of the `RelayETH` event
        vm.expectEmit(address(superchainETHBridge));
        emit RelayETH(_from, _to, _amount, _source);

        // Expect the call to the `mint` function in the `ETHLiquidity` contract
        vm.expectCall(Predeploys.ETH_LIQUIDITY, abi.encodeCall(IETHLiquidity.mint, (_amount)), 1);

        // Call the `RelayETH` function with the messenger caller
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        superchainETHBridge.relayETH(_from, _to, _amount);

        assertEq(_to.balance, _toBalanceBefore + _amount);

        // The relay consumed exactly its own amount of the source chain's cap
        assertEq(superchainETHBridge.netSent(_source), 0);
    }

    /// @notice A chain nobody has sent ETH to cannot extract anything, no matter what message it
    ///         claims to have exported. This is the lying-attester case the cap exists for.
    function testFuzz_relayETH_noNetFlow_reverts(uint256 _source, address _to, uint256 _amount) public {
        _amount = bound(_amount, 1, type(uint248).max - 1);
        vm.assume(_to != ZERO_ADDRESS);

        // Plenty of liquidity available: the only thing stopping the mint is the cap.
        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint128).max);

        vm.expectRevert(ISuperchainETHBridge.InsufficientNetFlow.selector);
        _relayFrom(_source, address(this), _to, _amount);
    }

    /// @notice Relaying more than was sent reverts, down to a single wei of overdraft.
    function testFuzz_relayETH_exceedsNetFlow_reverts(uint256 _source, uint256 _sent, uint256 _over) public {
        _sent = bound(_sent, 1, type(uint128).max);
        _over = bound(_over, 1, type(uint128).max);
        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint128).max);

        _sendToChain(_source, _sent);
        assertEq(superchainETHBridge.netSent(_source), _sent);

        vm.expectRevert(ISuperchainETHBridge.InsufficientNetFlow.selector);
        _relayFrom(_source, address(this), address(0xBEEF), _sent + _over);

        // A reverted relay consumes none of the cap.
        assertEq(superchainETHBridge.netSent(_source), _sent);
    }

    /// @notice Relaying exactly the sent amount succeeds and empties the cap; a second relay of any
    ///         nonzero amount then reverts. The counter cannot be decremented twice for the same
    ///         credit, which is the accounting half of replay protection (the messenger's
    ///         `successfulMessages` map is the other half and is untouched here).
    function testFuzz_relayETH_secondRelay_reverts(uint256 _source, uint256 _sent) public {
        _sent = bound(_sent, 1, type(uint128).max);
        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint128).max);

        _sendToChain(_source, _sent);
        _relayFrom(_source, address(this), address(0xBEEF), _sent);
        assertEq(superchainETHBridge.netSent(_source), 0);

        vm.expectRevert(ISuperchainETHBridge.InsufficientNetFlow.selector);
        _relayFrom(_source, address(this), address(0xBEEF), _sent);
    }

    /// @notice The cap draws down across many partial relays and stops at exactly the sent total.
    function test_relayETH_partialRelaysDrainExactly_succeeds() public {
        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint128).max);
        _sendToChain(CHAIN_P, 9 ether);

        _relayFrom(CHAIN_P, address(this), address(0xBEEF), 4 ether);
        assertEq(superchainETHBridge.netSent(CHAIN_P), 5 ether);

        _relayFrom(CHAIN_P, address(this), address(0xBEEF), 5 ether);
        assertEq(superchainETHBridge.netSent(CHAIN_P), 0);

        vm.expectRevert(ISuperchainETHBridge.InsufficientNetFlow.selector);
        _relayFrom(CHAIN_P, address(this), address(0xBEEF), 1 wei);
    }

    /// @notice Caps are per counterparty: sending to one chain grants no allowance to another.
    function test_relayETH_perChainCaps_succeeds() public {
        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint128).max);
        _sendToChain(CHAIN_P, 1 ether);

        // Chain B was sent nothing, so it can extract nothing even though P has room.
        vm.expectRevert(ISuperchainETHBridge.InsufficientNetFlow.selector);
        _relayFrom(CHAIN_B, address(this), address(0xBEEF), 1 ether);

        // P's own allowance is unaffected by B's failed attempt.
        assertEq(superchainETHBridge.netSent(CHAIN_P), 1 ether);
        _relayFrom(CHAIN_P, address(this), address(0xBEEF), 1 ether);
        assertEq(superchainETHBridge.netSent(CHAIN_P), 0);
        assertEq(superchainETHBridge.netSent(CHAIN_B), 0);
    }

    /// @notice A zero-amount relay is allowed at a zero cap: it mints nothing, so it cannot violate
    ///         the invariant, and rejecting it would break otherwise valid zero-value messages.
    function testFuzz_relayETH_zeroAmount_succeeds(uint256 _source) public {
        assertEq(superchainETHBridge.netSent(_source), 0);
        _relayFrom(_source, address(this), address(0xBEEF), 0);
        assertEq(superchainETHBridge.netSent(_source), 0);
    }
}

/// @title SuperchainETHBridge_Uncategorized_Test
/// @notice Tests the per-counterparty net-flow solvency cap as a whole-contract property: no chain can
///         ever cause more ETH to be minted on this chain than this chain voluntarily sent to it. See
///         `SPEC-ETH-NETFLOW-CAP.md` in the Silhouette project for the invariant.
contract SuperchainETHBridge_Uncategorized_Test is SuperchainETHBridge_TestInit {
    /// @notice At genesis every counterpart's cap is zero, so nothing is extractable from this chain
    ///         until ETH has actually been sent out. This assertion IS the security property.
    function testFuzz_netSent_genesis_succeeds(uint256 _chainId) public view {
        assertEq(superchainETHBridge.netSent(_chainId), 0);
    }

    /// @notice A round trip A->P->A->P leaves the accounting exact at every step: the cap is a
    ///         running net figure, not a one-shot allowance.
    function test_netSent_roundTrip_succeeds() public {
        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint128).max);

        // A -> P
        _sendToChain(CHAIN_P, 3 ether);
        assertEq(superchainETHBridge.netSent(CHAIN_P), 3 ether);

        // P -> A (the ETH comes home)
        _relayFrom(CHAIN_P, address(this), address(0xBEEF), 3 ether);
        assertEq(superchainETHBridge.netSent(CHAIN_P), 0);

        // A -> P again
        _sendToChain(CHAIN_P, 3 ether);
        assertEq(superchainETHBridge.netSent(CHAIN_P), 3 ether);

        // P -> A again, in two pieces
        _relayFrom(CHAIN_P, address(this), address(0xBEEF), 1 ether);
        _relayFrom(CHAIN_P, address(this), address(0xBEEF), 2 ether);
        assertEq(superchainETHBridge.netSent(CHAIN_P), 0);

        // And the chain is back to being able to extract nothing.
        vm.expectRevert(ISuperchainETHBridge.InsufficientNetFlow.selector);
        _relayFrom(CHAIN_P, address(this), address(0xBEEF), 1 wei);
    }

    /// @notice The lockbox invariant over an arbitrary interleaving: total ETH relayed in from a
    ///         chain never exceeds total ETH sent out to it, and every relay within the cap is
    ///         allowed. Sequence of (isSend, amount) steps driven by the fuzzer.
    function testFuzz_netSent_lockboxInvariant_succeeds(
        bool[16] calldata _isSend,
        uint64[16] calldata _amounts
    )
        public
    {
        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint128).max);

        uint256 _sent;
        uint256 _relayed;

        for (uint256 i = 0; i < 16; i++) {
            uint256 _amount = uint256(_amounts[i]);

            if (_isSend[i]) {
                _sendToChain(CHAIN_P, _amount);
                _sent += _amount;
            } else if (_amount <= _sent - _relayed) {
                // Within the cap: must be allowed.
                _relayFrom(CHAIN_P, address(this), address(0xBEEF), _amount);
                _relayed += _amount;
            } else {
                // Beyond the cap: must be refused.
                vm.expectRevert(ISuperchainETHBridge.InsufficientNetFlow.selector);
                _relayFrom(CHAIN_P, address(this), address(0xBEEF), _amount);
            }

            // The invariant, checked after every step.
            assertLe(_relayed, _sent);
            assertEq(superchainETHBridge.netSent(CHAIN_P), _sent - _relayed);
        }
    }
}
