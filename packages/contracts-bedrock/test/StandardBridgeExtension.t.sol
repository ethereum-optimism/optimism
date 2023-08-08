// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { StandardBridge } from "../src/universal/StandardBridge.sol";
import { StandardBridgeExtension } from "../src/universal/StandardBridgeExtension.sol";
import { Bridge_Initializer, CommonTest } from "./CommonTest.t.sol";

contract StandardBridgeExtensionTester is StandardBridgeExtension {
    constructor(
        address payable _messenger,
        address payable _otherBridge,
        address _localToken,
        address _remoteToken
    ) StandardBridgeExtension(_messenger, _otherBridge, _localToken, _remoteToken) {}

    function revertIfWrongERC20(address _localToken, address _remoteToken) onlyERC20(_localToken, _remoteToken) public {}
}

contract StandardBridgeExtension_Stateless_Test is CommonTest {
    StandardBridgeExtensionTester bridge;

    function setUp() public override {
        super.setUp();
        bridge = new StandardBridgeExtensionTester ({
            _messenger: payable(address(0)),
            _otherBridge: payable(address(0)),
            _localToken: address(0),
            _remoteToken: address(1)
        });
    }

    function test_revertIfWrongERC20() external {
        // correct token pair
        bridge.revertIfWrongERC20(address(0), address(1));

        // swapped
        vm.expectRevert();
        bridge.revertIfWrongERC20(address(1), address(0));

        vm.expectRevert();
        bridge.bridgeERC20(address(1), address(0), 0, 0, hex"");

        // just wrong
        vm.expectRevert();
        bridge.revertIfWrongERC20(address(1), address(2));

        vm.expectRevert();
        bridge.bridgeERC20To(address(1), address(0), address(0), 0, 0, hex"");
    }

    function test_revertWhenBridgingETH() external {
        vm.prank(alice, alice);

        // receive
        (bool success, ) = address(bridge).call{ value: 100 }(hex"");
        assertEq(success, false);

        // bridgeETH
        vm.expectRevert();
        bridge.bridgeETH(0, hex"");
        assertEq(success, false);

        // bridgeETHTo
        vm.expectRevert();
        bridge.bridgeETHTo(address(0), 0, hex"");
        assertEq(success, false);
    }
}

contract StandardBridgeExtension_Test is Bridge_Initializer {
    StandardBridgeExtensionTester bridge;

    function setUp() public override {
        super.setUp();
        bridge = new StandardBridgeExtensionTester ({
            _messenger: payable(L1Bridge.MESSENGER.address),
            _otherBridge: payable(L1Bridge.OTHER_BRIDGE.address),
            _localToken: address(L1Token),
            _remoteToken: address(L2Token)
        });
    }

    function test_bridge() external {
        vm.prank(alice);
        deal(address(L1Token), alice, 100000, true);
        L1Token.approve(address(bridge), type(uint256).max);

        // simply forwards to StandardBridge#bridgeERC20
        bridge.bridge(100, 0, hex"");
        vm.expectCall(
            address(bridge),
            abi.encodeWithSelector(StandardBridge.bridgeERC20.selector, address(L1Token), address(L2Token), 100, 0, hex"")
        );
        assertEq(bridge.deposits(address(L1Token), address(L2Token)), 100);

        // simply forwards to StandardBridge#bridgeERC20To
        bridge.bridgeTo(address(1), 100, 0, hex"");
        vm.expectCall(
            address(bridge),
            abi.encodeWithSelector(StandardBridge.bridgeERC20To.selector, address(L1Token), address(L2Token), address(1), 100, 0, hex"")
        );
        assertEq(bridge.deposits(address(L1Token), address(L2Token)), 200);
    }
}
