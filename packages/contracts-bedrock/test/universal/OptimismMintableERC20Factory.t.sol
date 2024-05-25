// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC20 } from "src/universal/OptimismMintableERC20.sol";
import { Bridge_Initializer } from "test/setup/Bridge_Initializer.sol";

contract OptimismMintableTokenFactory_Test is Bridge_Initializer {
    event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken);
    event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer);

    function test_bridge_succeeds() external {
        assertEq(address(l2OptimismMintableERC20Factory.BRIDGE()), address(l2StandardBridge));
    }

    function test_createStandardL2Token_succeeds() external {
        address remote = address(4);

        // Defaults to 18 decimals
        address local = calculateTokenAddress(remote, "Beep", "BOOP", 18);

        vm.expectEmit(true, true, true, true);
        emit StandardL2TokenCreated(remote, local);

        vm.expectEmit(true, true, true, true);
        emit OptimismMintableERC20Created(local, remote, alice);

        vm.prank(alice);
        address addr = l2OptimismMintableERC20Factory.createStandardL2Token(remote, "Beep", "BOOP");
        assertTrue(addr == local);
        assertTrue(OptimismMintableERC20(local).decimals() == 18);
    }

    function test_createStandardL2TokenWithDecimals_succeeds() external {
        address remote = address(4);
        address local = calculateTokenAddress(remote, "Beep", "BOOP", 6);

        vm.expectEmit(true, true, true, true);
        emit StandardL2TokenCreated(remote, local);

        vm.expectEmit(true, true, true, true);
        emit OptimismMintableERC20Created(local, remote, alice);

        vm.prank(alice);
        address addr = l2OptimismMintableERC20Factory.createOptimismMintableERC20WithDecimals(remote, "Beep", "BOOP", 6);
        assertTrue(addr == local);

        assertTrue(OptimismMintableERC20(local).decimals() == 6);
    }

    function test_createStandardL2Token_sameTwice_reverts() external {
        address remote = address(4);

        vm.prank(alice);
        l2OptimismMintableERC20Factory.createStandardL2Token(remote, "Beep", "BOOP");

        vm.expectRevert();

        vm.prank(alice);
        l2OptimismMintableERC20Factory.createStandardL2Token(remote, "Beep", "BOOP");
    }

    function test_createStandardL2Token_remoteIsZero_reverts() external {
        address remote = address(0);
        vm.expectRevert("OptimismMintableERC20Factory: must provide remote token address");
        l2OptimismMintableERC20Factory.createStandardL2Token(remote, "Beep", "BOOP");
    }

    function calculateTokenAddress(
        address _remote,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        internal
        view
        returns (address)
    {
        bytes memory constructorArgs = abi.encode(address(l2StandardBridge), _remote, _name, _symbol, _decimals);
        bytes memory bytecode = abi.encodePacked(type(OptimismMintableERC20).creationCode, constructorArgs);
        bytes32 salt = keccak256(abi.encode(_remote, _name, _symbol, _decimals));
        bytes32 hash = keccak256(
            abi.encodePacked(bytes1(0xff), address(l2OptimismMintableERC20Factory), salt, keccak256(bytecode))
        );
        return address(uint160(uint256(hash)));
    }
}
