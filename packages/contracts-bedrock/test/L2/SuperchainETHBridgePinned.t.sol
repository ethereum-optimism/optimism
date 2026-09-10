// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";

// Interfaces
import { ISuperchainETHBridgePinned } from "interfaces/L2/ISuperchainETHBridgePinned.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";

/// @title SuperchainETHBridgePinned_TestInit
/// @notice Reusable test initialization for `SuperchainETHBridgePinned` tests. Stands in for the
///         private chain's own genesis: the pinned implementation behind the predeploy, with
///         `homeChainId` written by allocation rather than by any contract code.
abstract contract SuperchainETHBridgePinned_TestInit is CommonTest {
    /// @notice The one chain this bridge may exchange ETH with.
    uint256 internal constant HOME_CHAIN = 901;

    /// @notice Some other public chain in the dependency set.
    uint256 internal constant OTHER_CHAIN = 8453;

    /// @notice Storage slot of `homeChainId`: slot 0 is the inherited `netSent` mapping.
    bytes32 internal constant HOME_CHAIN_ID_SLOT = bytes32(uint256(1));

    ISuperchainETHBridgePinned internal bridge;

    function setUp() public virtual override {
        super.enableInterop();
        super.setUp();

        vm.etch(
            address(superchainETHBridge), vm.getDeployedCode("SuperchainETHBridgePinned.sol:SuperchainETHBridgePinned")
        );
        vm.etch(address(ethLiquidity), vm.getDeployedCode("ETHLiquidity.sol:ETHLiquidity"));
        bridge = ISuperchainETHBridgePinned(address(superchainETHBridge));

        // This is the whole configuration mechanism: one storage word placed by genesis. There is no
        // setter and no owner, so nothing else in these tests could have written it either.
        vm.store(address(bridge), HOME_CHAIN_ID_SLOT, bytes32(HOME_CHAIN));

        vm.deal(Predeploys.ETH_LIQUIDITY, type(uint128).max);
    }

    /// @notice Undoes the genesis configuration, leaving the bridge unconfigured.
    function _unconfigure() internal {
        vm.store(address(bridge), HOME_CHAIN_ID_SLOT, bytes32(0));
    }

    /// @notice Sends `_amount` to `_chainId` from a funded EOA, with the messenger mocked out.
    function _send(uint256 _chainId, uint256 _amount) internal {
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeWithSelector(IL2ToL2CrossDomainMessenger.sendMessage.selector),
            abi.encode(bytes32(0))
        );
        address _funder = address(uint160(uint256(keccak256("pinned.funder"))));
        vm.deal(_funder, _amount);
        vm.prank(_funder);
        bridge.sendETH{ value: _amount }(address(uint160(uint256(keccak256("pinned.to")))), _chainId);
        vm.clearMockedCalls();
    }

    /// @notice Relays `_amount` as if it arrived from `_source`.
    function _relayFrom(uint256 _source, uint256 _amount) internal {
        vm.mockCall(
            Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeCall(IL2ToL2CrossDomainMessenger.crossDomainMessageContext, ()),
            abi.encode(address(bridge), _source)
        );
        vm.prank(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bridge.relayETH(address(this), address(0xBEEF), _amount);
    }

    /// @notice Writes `netSent[_chainId]` directly, to prove the home-chain pin holds even against a
    ///         cap that has somehow been credited for a non-home chain.
    function _forceNetSent(uint256 _chainId, uint256 _amount) internal {
        vm.store(address(bridge), keccak256(abi.encode(_chainId, uint256(0))), bytes32(_amount));
    }
}

/// @title SuperchainETHBridgePinned_SendETH_Test
/// @notice Tests the outbound half of the home-chain pin.
contract SuperchainETHBridgePinned_SendETH_Test is SuperchainETHBridgePinned_TestInit {
    /// @notice Sending to the home chain works and credits the home chain's cap.
    function test_sendETH_homeChain_succeeds() public {
        _send(HOME_CHAIN, 2 ether);
        assertEq(bridge.netSent(HOME_CHAIN), 2 ether);
    }

    /// @notice Sending anywhere else reverts, and reverts BEFORE the ETH is burned: the user keeps
    ///         their funds, so a misroute originating on this chain is impossible rather than merely
    ///         unrecoverable.
    function testFuzz_sendETH_nonHomeChain_reverts(uint256 _chainId, uint256 _amount) public {
        vm.assume(_chainId != HOME_CHAIN);
        _amount = bound(_amount, 1, type(uint128).max);

        address _sender = address(uint160(uint256(keccak256("pinned.sender"))));
        vm.deal(_sender, _amount);

        vm.expectRevert(ISuperchainETHBridgePinned.NotHomeChain.selector);
        vm.prank(_sender);
        bridge.sendETH{ value: _amount }(address(0xBEEF), _chainId);

        // No burn happened and no cap was credited.
        assertEq(_sender.balance, _amount);
        assertEq(bridge.netSent(_chainId), 0);
    }

    /// @notice Chain ID zero is never a valid destination, even though it is also never a real chain.
    function test_sendETH_chainIdZero_reverts() public {
        vm.deal(address(this), 1 ether);
        vm.expectRevert(ISuperchainETHBridgePinned.NotHomeChain.selector);
        bridge.sendETH{ value: 1 ether }(address(0xBEEF), 0);
    }

    /// @notice With no home chain configured, nothing can be sent anywhere — a genesis mistake makes
    ///         the bridge inert rather than unpinned.
    function testFuzz_sendETH_unconfigured_reverts(uint256 _chainId) public {
        _unconfigure();
        vm.deal(address(this), 1 ether);
        vm.expectRevert(ISuperchainETHBridgePinned.NotHomeChain.selector);
        bridge.sendETH{ value: 1 ether }(address(0xBEEF), _chainId);
    }
}

/// @title SuperchainETHBridgePinned_RelayETH_Test
/// @notice Tests the inbound half of the home-chain pin.
contract SuperchainETHBridgePinned_RelayETH_Test is SuperchainETHBridgePinned_TestInit {
    /// @notice ETH coming back from the home chain relays normally, drawing down the cap.
    function test_relayETH_homeChain_succeeds() public {
        _send(HOME_CHAIN, 2 ether);

        uint256 _before = address(0xBEEF).balance;
        _relayFrom(HOME_CHAIN, 2 ether);

        assertEq(address(0xBEEF).balance, _before + 2 ether);
        assertEq(bridge.netSent(HOME_CHAIN), 0);
    }

    /// @notice A relay claiming any other source chain is refused — even when its cap has been
    ///         credited, which proves the pin is an independent check and not a restatement of the
    ///         net-flow cap.
    function testFuzz_relayETH_nonHomeSource_reverts(uint256 _source, uint256 _amount) public {
        vm.assume(_source != HOME_CHAIN);
        _amount = bound(_amount, 1, type(uint128).max);

        _forceNetSent(_source, _amount);

        vm.expectRevert(ISuperchainETHBridgePinned.NotHomeChain.selector);
        _relayFrom(_source, _amount);

        // The forced credit was not consumed: the relay never reached the accounting.
        assertEq(bridge.netSent(_source), _amount);
    }

    /// @notice A source chain with no credit is refused by the pin, which runs first. Both checks
    ///         would refuse it; the assertion is that the outer, intent-stating one fires.
    function testFuzz_relayETH_nonHomeSourceNoCredit_reverts(uint256 _source) public {
        vm.assume(_source != HOME_CHAIN);
        vm.expectRevert(ISuperchainETHBridgePinned.NotHomeChain.selector);
        _relayFrom(_source, 1 ether);
    }

    /// @notice The inherited cap still governs relays from the home chain: the pin does not weaken it.
    function test_relayETH_exceedsCap_reverts() public {
        _send(HOME_CHAIN, 1 ether);

        vm.expectRevert(ISuperchainETHBridgePinned.InsufficientNetFlow.selector);
        _relayFrom(HOME_CHAIN, 1 ether + 1 wei);

        _relayFrom(HOME_CHAIN, 1 ether);
        vm.expectRevert(ISuperchainETHBridgePinned.InsufficientNetFlow.selector);
        _relayFrom(HOME_CHAIN, 1 wei);
    }

    /// @notice A non-messenger caller still gets `Unauthorized`, not the messenger's not-entered
    ///         error: the override reproduces the base's guard before reading message context.
    function testFuzz_relayETH_notMessenger_reverts(address _caller) public {
        vm.assume(_caller != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        vm.expectRevert(Unauthorized.selector);
        vm.prank(_caller);
        bridge.relayETH(_caller, address(0xBEEF), 1 ether);
    }

    /// @notice With no home chain configured, nothing can be relayed from anywhere, even with a
    ///         credited cap.
    function testFuzz_relayETH_unconfigured_reverts(uint256 _source) public {
        _unconfigure();
        _forceNetSent(_source, 1 ether);
        vm.expectRevert(ISuperchainETHBridgePinned.NotHomeChain.selector);
        _relayFrom(_source, 1 ether);
    }
}

/// @title SuperchainETHBridgePinned_Uncategorized_Test
/// @notice Tests the genesis-only configuration and the property the pin exists to create: this
///         chain's ETH has exactly one counterpart, so there is exactly one running net figure for it
///         anywhere in the dependency set.
contract SuperchainETHBridgePinned_Uncategorized_Test is SuperchainETHBridgePinned_TestInit {
    /// @notice The configuration word is where genesis allocation must put it, and it reads back
    ///         through the getter. Pins the storage layout the alloc depends on.
    function test_homeChainId_genesisSlot_succeeds() public view {
        assertEq(uint256(vm.load(address(bridge), HOME_CHAIN_ID_SLOT)), HOME_CHAIN);
        assertEq(bridge.homeChainId(), HOME_CHAIN);
    }

    /// @notice The inherited cap mapping still occupies slot 0, so the alloc's slot 1 is not
    ///         colliding with a `netSent` entry.
    function test_netSent_slotZero_succeeds() public {
        _forceNetSent(OTHER_CHAIN, 7 ether);
        assertEq(bridge.netSent(OTHER_CHAIN), 7 ether);
        assertEq(bridge.homeChainId(), HOME_CHAIN);
    }

    /// @notice Version reflects the variant, per the repo's variant convention.
    function test_version_pinnedVariant_succeeds() public view {
        assertEq(bridge.version(), "1.1.0+home-pinned");
    }

    /// @notice Deposits and withdrawals both flow through the home chain only, and the home chain's
    ///         counter tracks the running net at every step. Because this chain cannot send to, or
    ///         relay from, anyone else, the home chain's `netSent[thisChain]` is the global figure for
    ///         all ETH this chain holds — there is nowhere else for it to have come from or gone.
    function test_netSent_singleCounter_succeeds() public {
        _send(HOME_CHAIN, 5 ether);
        assertEq(bridge.netSent(HOME_CHAIN), 5 ether);

        _relayFrom(HOME_CHAIN, 2 ether);
        assertEq(bridge.netSent(HOME_CHAIN), 3 ether);

        _send(HOME_CHAIN, 1 ether);
        assertEq(bridge.netSent(HOME_CHAIN), 4 ether);

        _relayFrom(HOME_CHAIN, 4 ether);
        assertEq(bridge.netSent(HOME_CHAIN), 0);
    }

    /// @notice No counter other than the home chain's can ever become nonzero through contract code,
    ///         because the only writer of `netSent` is `sendETH`, and `sendETH` only accepts the home
    ///         chain.
    function testFuzz_netSent_noOtherCounter_succeeds(uint256 _other) public {
        vm.assume(_other != HOME_CHAIN);

        _send(HOME_CHAIN, 3 ether);
        _relayFrom(HOME_CHAIN, 1 ether);

        assertEq(bridge.netSent(_other), 0);

        // And the only way to move it is refused.
        vm.deal(address(this), 1 ether);
        vm.expectRevert(ISuperchainETHBridgePinned.NotHomeChain.selector);
        bridge.sendETH{ value: 1 ether }(address(0xBEEF), _other);
        assertEq(bridge.netSent(_other), 0);
    }

    /// @notice The running figure equals sent minus relayed across an arbitrary interleaving, with
    ///         every step forced through the single home counter.
    function testFuzz_netSent_singleCounterNet_succeeds(
        bool[12] calldata _isSend,
        uint64[12] calldata _amounts
    )
        public
    {
        uint256 _sent;
        uint256 _relayed;

        for (uint256 i = 0; i < 12; i++) {
            uint256 _amount = uint256(_amounts[i]);

            if (_isSend[i]) {
                _send(HOME_CHAIN, _amount);
                _sent += _amount;
            } else if (_amount <= _sent - _relayed) {
                _relayFrom(HOME_CHAIN, _amount);
                _relayed += _amount;
            } else {
                vm.expectRevert(ISuperchainETHBridgePinned.InsufficientNetFlow.selector);
                _relayFrom(HOME_CHAIN, _amount);
            }

            assertLe(_relayed, _sent);
            assertEq(bridge.netSent(HOME_CHAIN), _sent - _relayed);
        }
    }
}
