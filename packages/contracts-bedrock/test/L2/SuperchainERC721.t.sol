// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Contracts
import { SuperchainERC721 } from "src/L2/SuperchainERC721.sol";
import { MockSuperchainERC721Implementation } from "test/mocks/SuperchainERC721Implementation.sol";
import { ERC721 } from "@solady-v0.0.245/tokens/ERC721.sol";

// Interfaces
import { ICrosschainERC721, IERC165 } from "interfaces/L2/ICrosschainERC721.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import { ISuperchainERC721 } from "interfaces/L2/ISuperchainERC721.sol";

/// @title SuperchainERC721_TestInit
/// @notice Reusable test initialization for `SuperchainERC721` tests.
contract SuperchainERC721_TestInit is CommonTest {
    address internal constant ZERO_ADDRESS = address(0);

    SuperchainERC721 public superchainERC721;

    // Events
    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);
    event CrosschainMint(address indexed to, uint256 tokenId, address indexed sender);
    event CrosschainBurn(address indexed from, uint256 tokenId, address indexed sender);

    /// @notice Sets up the test suite.
    function setUp() public override {
        useInteropOverride = true;
        super.setUp();
        superchainERC721 = new MockSuperchainERC721Implementation();
    }

    /// @notice Helper function to setup a mock and expect a call to it.
    function _mockAndExpect(address _receiver, bytes memory _calldata, bytes memory _returned) internal {
        vm.mockCall(_receiver, _calldata, _returned);
        vm.expectCall(_receiver, _calldata);
    }
}

/// @title SuperchainERC721_CrosschainMint_Test
/// @notice Tests the `crosschainMint` function of the `SuperchainERC721` contract.
contract SuperchainERC721_CrosschainMint_Test is SuperchainERC721_TestInit {
    /// @notice Tests the `crosschainMint` function reverts when the caller is not the bridge.
    function testFuzz_crosschainMint_callerNotBridge_reverts(address _caller, address _to, uint256 _tokenId) public {
        // Ensure the caller is not the bridge
        vm.assume(_caller != Predeploys.SUPERCHAIN_NFT_BRIDGE);

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(ISuperchainERC721.Unauthorized.selector);

        // Call the `mint` function with the non-bridge caller
        vm.prank(_caller);
        superchainERC721.crosschainMint(_to, _tokenId);
    }

    /// @notice Tests the `crosschainMint` succeeds and emits the `CrosschainMint` event.
    function testFuzz_crosschainMint_succeeds(address _to, uint256 _tokenId) public {
        // Ensure `_to` is not the zero address
        vm.assume(_to != ZERO_ADDRESS);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(address(superchainERC721));
        emit Transfer(ZERO_ADDRESS, _to, _tokenId);

        // Look for the emit of the `CrosschainMint` event
        vm.expectEmit(address(superchainERC721));
        emit CrosschainMint(_to, _tokenId, Predeploys.SUPERCHAIN_NFT_BRIDGE);

        // Call the `mint` function with the bridge caller
        vm.prank(Predeploys.SUPERCHAIN_NFT_BRIDGE);
        superchainERC721.crosschainMint(_to, _tokenId);

        // Check the token was minted to `_to`
        assertEq(superchainERC721.ownerOf(_tokenId), _to);
    }
}

/// @title SuperchainERC721_CrosschainBurn_Test
/// @notice Tests the `crosschainBurn` function of the `SuperchainERC721` contract.
contract SuperchainERC721_CrosschainBurn_Test is SuperchainERC721_TestInit {
    /// @notice Tests the `crosschainBurn` function reverts when the caller is not the bridge.
    function testFuzz_crosschainBurn_callerNotBridge_reverts(address _caller, address _from, uint256 _tokenId) public {
        // Ensure the caller is not the bridge
        vm.assume(_caller != Predeploys.SUPERCHAIN_NFT_BRIDGE);

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(ISuperchainERC721.Unauthorized.selector);

        // Call the `burn` function with the non-bridge caller
        vm.prank(_caller);
        superchainERC721.crosschainBurn(_from, _tokenId);
    }

    /// @notice Tests the `crosschainBurn` burns the token with `_tokenId` and emits the
    ///         `CrosschainBurn` event.
    function testFuzz_crosschainBurn_succeeds(address _from, uint256 _tokenId) public {
        // Ensure `_from` is not the zero address
        vm.assume(_from != ZERO_ADDRESS);

        // Mint some tokens to `_from` so then they can be burned
        vm.prank(Predeploys.SUPERCHAIN_NFT_BRIDGE);
        superchainERC721.crosschainMint(_from, _tokenId);

        // Look for the emit of the `Transfer` event
        vm.expectEmit(address(superchainERC721));
        emit Transfer(_from, ZERO_ADDRESS, _tokenId);

        // Look for the emit of the `CrosschainBurn` event
        vm.expectEmit(address(superchainERC721));
        emit CrosschainBurn(_from, _tokenId, Predeploys.SUPERCHAIN_NFT_BRIDGE);

        // Call the `burn` function with the bridge caller
        vm.prank(Predeploys.SUPERCHAIN_NFT_BRIDGE);
        superchainERC721.crosschainBurn(_from, _tokenId);

        // Check the token was burnt
        vm.expectRevert(ERC721.TokenDoesNotExist.selector);
        superchainERC721.ownerOf(_tokenId);
    }
}

/// @title SuperchainERC721_SupportsInterfaces_Test
/// @notice Tests the `supportsInterface` function of the `SuperchainERC721` contract.
contract SuperchainERC721_SupportsInterfaces_Test is SuperchainERC721_TestInit {
    /// @notice Tests that the `supportsInterface` function returns true for the `ICrosschainERC721`
    ///         interface.
    function test_supportInterface_succeeds() public view {
        assertTrue(superchainERC721.supportsInterface(type(IERC165).interfaceId));
        assertTrue(superchainERC721.supportsInterface(type(ICrosschainERC721).interfaceId));
        assertTrue(superchainERC721.supportsInterface(type(IERC721).interfaceId));
    }

    /// @notice Tests that the `supportsInterface` function returns false for any other interface
    ///         than the `ICrosschainERC721` one.
    function testFuzz_supportInterface_works(bytes4 _interfaceId) public view {
        vm.assume(_interfaceId != type(IERC165).interfaceId);
        vm.assume(_interfaceId != type(ICrosschainERC721).interfaceId);
        vm.assume(_interfaceId != type(IERC721).interfaceId);
        assertFalse(superchainERC721.supportsInterface(_interfaceId));
    }
}
