// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { OptimismMintableERC721 } from "src/L2/OptimismMintableERC721.sol";

/// @title OptimismMintableERC721Factory_TestInit
/// @notice Reusable test initialization for `OptimismMintableERC721Factory` tests.
abstract contract OptimismMintableERC721Factory_TestInit is CommonTest {
    event OptimismMintableERC721Created(address indexed localToken, address indexed remoteToken, address deployer);

    function calculateTokenAddress(
        address _remote,
        string memory _name,
        string memory _symbol
    )
        internal
        view
        returns (address)
    {
        bytes memory constructorArgs =
            abi.encode(address(l2ERC721Bridge), deploy.cfg().l1ChainID(), _remote, _name, _symbol);
        bytes memory bytecode = abi.encodePacked(type(OptimismMintableERC721).creationCode, constructorArgs);
        bytes32 salt = keccak256(abi.encode(_remote, _name, _symbol));
        bytes32 hash = keccak256(
            abi.encodePacked(bytes1(0xff), address(l2OptimismMintableERC721Factory), salt, keccak256(bytecode))
        );
        return address(uint160(uint256(hash)));
    }
}

/// @title OptimismMintableERC721Factory_Constructor_Test
/// @notice Tests the `constructor` of the `OptimismMintableERC721Factory` contract.
contract OptimismMintableERC721Factory_Constructor_Test is OptimismMintableERC721Factory_TestInit {
    /// @notice Tests that the constructor sets the correct values.
    function test_constructor_succeeds() external view {
        assertEq(l2OptimismMintableERC721Factory.BRIDGE(), address(l2ERC721Bridge));
        assertEq(l2OptimismMintableERC721Factory.bridge(), address(l2ERC721Bridge));
        assertEq(l2OptimismMintableERC721Factory.REMOTE_CHAIN_ID(), deploy.cfg().l1ChainID());
        assertEq(l2OptimismMintableERC721Factory.remoteChainID(), deploy.cfg().l1ChainID());
    }
}

/// @title OptimismMintableERC721Factory_CreateOptimismMintableERC721_Test
/// @notice Tests the `createOptimismMintableERC721` function of the
///         `OptimismMintableERC721Factory` contract.
contract OptimismMintableERC721Factory_CreateOptimismMintableERC721_Test is OptimismMintableERC721Factory_TestInit {
    /// @notice Tests that the `createOptimismMintableERC721` function succeeds.
    function test_createOptimismMintableERC721_succeeds() external {
        address remote = address(1234);
        address local = calculateTokenAddress(address(1234), "L2Token", "L2T");

        // Expect a token creation event.
        vm.expectEmit(address(l2OptimismMintableERC721Factory));
        emit OptimismMintableERC721Created(local, remote, alice);

        // Create the token.
        vm.prank(alice);
        OptimismMintableERC721 created = OptimismMintableERC721(
            l2OptimismMintableERC721Factory.createOptimismMintableERC721(remote, "L2Token", "L2T")
        );

        // Token address should be correct.
        assertEq(address(created), local);

        // Should be marked as created by the factory.
        assertTrue(l2OptimismMintableERC721Factory.isOptimismMintableERC721(address(created)));

        // Token should've been constructed correctly.
        assertEq(created.name(), "L2Token");
        assertEq(created.symbol(), "L2T");
        assertEq(created.REMOTE_TOKEN(), remote);
        assertEq(created.BRIDGE(), address(l2ERC721Bridge));
        assertEq(created.REMOTE_CHAIN_ID(), deploy.cfg().l1ChainID());
    }

    /// @notice Tests that the `createOptimismMintableERC721` function reverts if the same token is
    ///         created twice.
    function test_createOptimismMintableERC721_sameTwice_reverts() external {
        address remote = address(1234);

        vm.prank(alice);
        l2OptimismMintableERC721Factory.createOptimismMintableERC721(remote, "L2Token", "L2T");

        vm.expectRevert(bytes(""));

        vm.prank(alice);
        l2OptimismMintableERC721Factory.createOptimismMintableERC721(remote, "L2Token", "L2T");
    }

    /// @notice Tests that the `createOptimismMintableERC721` function reverts if the remote token
    ///         address is zero.
    function test_createOptimismMintableERC721_zeroRemoteToken_reverts() external {
        // Try to create a token with a zero remote token address.
        vm.expectRevert("OptimismMintableERC721Factory: L1 token address cannot be address(0)");
        l2OptimismMintableERC721Factory.createOptimismMintableERC721(address(0), "L2Token", "L2T");
    }
}
