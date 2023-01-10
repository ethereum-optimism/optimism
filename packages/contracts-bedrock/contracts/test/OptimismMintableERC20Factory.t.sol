// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Bridge_Initializer } from "./CommonTest.t.sol";
import { LibRLP } from "./RLP.t.sol";

contract OptimismMintableTokenFactory_Test is Bridge_Initializer {
    event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken);
    event OptimismMintableERC20Created(
        address indexed localToken,
        address indexed remoteToken,
        address deployer
    );

    function setUp() public override {
        super.setUp();
    }

    function test_bridge_succeeds() external {
        assertEq(address(L2TokenFactory.BRIDGE()), address(L2Bridge));
    }

    function test_createStandardL2Token_succeeds() external {
        address remote = address(4);
        address local = LibRLP.computeAddress(address(L2TokenFactory), 2);

        vm.expectEmit(true, true, true, true);
        emit StandardL2TokenCreated(remote, local);

        vm.expectEmit(true, true, true, true);
        emit OptimismMintableERC20Created(local, remote, alice);

        vm.prank(alice);
        L2TokenFactory.createStandardL2Token(remote, "Beep", "BOOP");
    }

    function test_createStandardL2Token_sameTwice_succeeds() external {
        address remote = address(4);

        vm.prank(alice);
        L2TokenFactory.createStandardL2Token(remote, "Beep", "BOOP");

        address local = LibRLP.computeAddress(address(L2TokenFactory), 3);

        vm.expectEmit(true, true, true, true);
        emit StandardL2TokenCreated(remote, local);

        vm.expectEmit(true, true, true, true);
        emit OptimismMintableERC20Created(local, remote, alice);

        vm.prank(alice);
        L2TokenFactory.createStandardL2Token(remote, "Beep", "BOOP");
    }

    function test_createStandardL2Token_remoteIsZero_succeeds() external {
        address remote = address(0);
        vm.expectRevert("OptimismMintableERC20Factory: must provide remote token address");
        L2TokenFactory.createStandardL2Token(remote, "Beep", "BOOP");
    }
}
