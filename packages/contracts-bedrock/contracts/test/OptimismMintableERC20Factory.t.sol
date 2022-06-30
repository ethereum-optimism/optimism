// SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Bridge_Initializer } from "./CommonTest.t.sol";
import { LibRLP } from "./Lib_RLP.t.sol";

contract OptimismMintableERC20Factory_Test is Bridge_Initializer {
    event StandardL2TokenCreated(address indexed _remoteToken, address indexed _localToken);
    event OptimismMintableERC20Created(
        address indexed _localToken,
        address indexed _remoteToken,
        address _deployer
    );

    function setUp() public override {
        super.setUp();
    }

    function test_bridge() external {
        assertEq(address(L2TokenFactory.bridge()), address(L2Bridge));
    }

    function test_createStandardL2Token() external {
        address remote = address(4);
        address local = LibRLP.computeAddress(address(L2TokenFactory), 1);

        vm.expectEmit(true, true, true, true);
        emit StandardL2TokenCreated(
            remote,
            local
        );

        vm.expectEmit(true, true, true, true);
        emit OptimismMintableERC20Created(
            remote,
            local,
            alice
        );

        vm.prank(alice);
        L2TokenFactory.createStandardL2Token(remote, "Beep", "BOOP");
    }

    function test_createStandardL2TokenSameTwice() external {
        address remote = address(4);

        vm.prank(alice);
        L2TokenFactory.createStandardL2Token(remote, "Beep", "BOOP");

        address local = LibRLP.computeAddress(address(L2TokenFactory), 2);

        vm.expectEmit(true, true, true, true);
        emit StandardL2TokenCreated(
            remote,
            local
        );

        vm.expectEmit(true, true, true, true);
        emit OptimismMintableERC20Created(
            remote,
            local,
            alice
        );

        vm.prank(alice);
        L2TokenFactory.createStandardL2Token(remote, "Beep", "BOOP");
    }

    function test_createStandardL2TokenShouldRevertIfRemoteIsZero() external {
        address remote = address(0);
        vm.expectRevert("OptimismMintableERC20Factory: L1 token address cannot be address(0)");
        L2TokenFactory.createStandardL2Token(remote, "Beep", "BOOP");
    }
}
