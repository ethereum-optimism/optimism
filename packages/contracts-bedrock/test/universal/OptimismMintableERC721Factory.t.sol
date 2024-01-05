// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import { Bridge_Initializer } from "test/setup/Bridge_Initializer.sol";
import { OptimismMintableERC721 } from "src/universal/OptimismMintableERC721.sol";
import { OptimismMintableERC721Factory } from "src/universal/OptimismMintableERC721Factory.sol";

contract OptimismMintableERC721Factory_Test is Bridge_Initializer {
    OptimismMintableERC721Factory internal factory;

    event OptimismMintableERC721Created(address indexed localToken, address indexed remoteToken, address deployer);

    function setUp() public override {
        super.setUp();

        // Set up the token pair.
        factory = new OptimismMintableERC721Factory(address(l2ERC721Bridge), 1);

        // Label the addresses for nice traces.
        vm.label(address(factory), "OptimismMintableERC721Factory");
    }

    function test_constructor_succeeds() external {
        assertEq(factory.BRIDGE(), address(l2ERC721Bridge));
        assertEq(factory.REMOTE_CHAIN_ID(), 1);
    }

    function test_createOptimismMintableERC721_succeeds() external {
        address remote = address(1234);
        address local = calculateTokenAddress(address(1234), "L2Token", "L2T");

        // Expect a token creation event.
        vm.expectEmit(true, true, true, true);
        emit OptimismMintableERC721Created(local, remote, alice);

        // Create the token.
        vm.prank(alice);
        OptimismMintableERC721 created =
            OptimismMintableERC721(factory.createOptimismMintableERC721(remote, "L2Token", "L2T"));

        // Token address should be correct.
        assertEq(address(created), local);

        // Should be marked as created by the factory.
        assertEq(factory.isOptimismMintableERC721(address(created)), true);

        // Token should've been constructed correctly.
        assertEq(created.name(), "L2Token");
        assertEq(created.symbol(), "L2T");
        assertEq(created.REMOTE_TOKEN(), remote);
        assertEq(created.BRIDGE(), address(l2ERC721Bridge));
        assertEq(created.REMOTE_CHAIN_ID(), 1);
    }

    function test_createOptimismMintableERC721_sameTwice_reverts() external {
        address remote = address(1234);

        vm.prank(alice);
        factory.createOptimismMintableERC721(remote, "L2Token", "L2T");

        vm.expectRevert();

        vm.prank(alice);
        factory.createOptimismMintableERC721(remote, "L2Token", "L2T");
    }

    function test_createOptimismMintableERC721_zeroRemoteToken_reverts() external {
        // Try to create a token with a zero remote token address.
        vm.expectRevert("OptimismMintableERC721Factory: L1 token address cannot be address(0)");
        factory.createOptimismMintableERC721(address(0), "L2Token", "L2T");
    }

    function calculateTokenAddress(
        address _remote,
        string memory _name,
        string memory _symbol
    )
        internal
        view
        returns (address)
    {
        bytes memory constructorArgs = abi.encode(address(l2ERC721Bridge), 1, _remote, _name, _symbol);
        bytes memory bytecode = abi.encodePacked(type(OptimismMintableERC721).creationCode, constructorArgs);
        bytes32 salt = keccak256(abi.encode(_remote, _name, _symbol));
        bytes32 hash = keccak256(abi.encodePacked(bytes1(0xff), address(factory), salt, keccak256(bytecode)));
        return address(uint160(uint256(hash)));
    }
}
