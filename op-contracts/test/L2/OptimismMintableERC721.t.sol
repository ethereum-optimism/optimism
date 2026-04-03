// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC721, IERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import { IERC721Enumerable } from "@openzeppelin/contracts/token/ERC721/extensions/ERC721Enumerable.sol";
import { IERC721Metadata } from "@openzeppelin/contracts/token/ERC721/extensions/IERC721Metadata.sol";
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { OptimismMintableERC721, IOptimismMintableERC721 } from "src/L2/OptimismMintableERC721.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

/// @title OptimismMintableERC721_TestInit
/// @notice Reusable test initialization for `OptimismMintableERC721` tests.
abstract contract OptimismMintableERC721_TestInit is CommonTest {
    ERC721 internal L1NFT;
    OptimismMintableERC721 internal L2NFT;

    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);

    event Mint(address indexed account, uint256 tokenId);

    event Burn(address indexed account, uint256 tokenId);

    function setUp() public override {
        super.setUp();

        // Set up the token pair.
        L1NFT = new ERC721("L1NFT", "L1T");
        L2NFT = new OptimismMintableERC721(address(l2ERC721Bridge), 1, address(L1NFT), "L2NFT", "L2T");

        // Label the addresses for nice traces.
        vm.label(address(L1NFT), "L1ERC721Token");
        vm.label(address(L2NFT), "L2ERC721Token");
    }
}

/// @title OptimismMintableERC721_Constructor_Test
/// @notice Tests the `constructor` of the `OptimismMintableERC721` contract.
contract OptimismMintableERC721_Constructor_Test is OptimismMintableERC721_TestInit {
    /// @notice Tests that the constructor initializes state variables correctly with valid inputs.
    function testFuzz_constructor_validParams_succeeds(uint256 _remoteChainId) external {
        vm.assume(_remoteChainId != 0);
        OptimismMintableERC721 nft =
            new OptimismMintableERC721(address(l2ERC721Bridge), _remoteChainId, address(L1NFT), "L2NFT", "L2T");
        assertEq(nft.name(), "L2NFT");
        assertEq(nft.symbol(), "L2T");
        assertEq(nft.remoteToken(), address(L1NFT));
        assertEq(nft.bridge(), address(l2ERC721Bridge));
        assertEq(nft.remoteChainId(), _remoteChainId);
        assertEq(nft.REMOTE_TOKEN(), address(L1NFT));
        assertEq(nft.BRIDGE(), address(l2ERC721Bridge));
        assertEq(nft.REMOTE_CHAIN_ID(), _remoteChainId);
    }

    /// @notice Tests that the constructor reverts when the bridge address is zero.
    function test_constructor_bridgeAsAddress0_reverts() external {
        vm.expectRevert("OptimismMintableERC721: bridge cannot be address(0)");
        L2NFT = new OptimismMintableERC721(address(0), 1, address(L1NFT), "L2NFT", "L2T");
    }

    /// @notice Tests that the constructor reverts when the remote chain ID is zero.
    function test_constructor_remoteChainId0_reverts() external {
        vm.expectRevert("OptimismMintableERC721: remote chain id cannot be zero");
        L2NFT = new OptimismMintableERC721(address(l2ERC721Bridge), 0, address(L1NFT), "L2NFT", "L2T");
    }

    /// @notice Tests that the constructor reverts when the remote token address is zero.
    function test_constructor_remoteTokenAsAddress0_reverts() external {
        vm.expectRevert("OptimismMintableERC721: remote token cannot be address(0)");
        L2NFT = new OptimismMintableERC721(address(l2ERC721Bridge), 1, address(0), "L2NFT", "L2T");
    }
}

/// @title OptimismMintableERC721_SafeMint_Test
/// @notice Tests the `safeMint` function of the `OptimismMintableERC721` contract.
contract OptimismMintableERC721_SafeMint_Test is OptimismMintableERC721_TestInit {
    /// @notice Tests that the `safeMint` function successfully mints a token when called by the
    ///         bridge.
    function testFuzz_safeMint_validParams_succeeds(address _to, uint256 _tokenId) external {
        vm.assume(_to != address(0));
        vm.assume(_to.code.length == 0);

        // Expect a transfer event.
        vm.expectEmit(true, true, true, true);
        emit Transfer(address(0), _to, _tokenId);

        // Expect a mint event.
        vm.expectEmit(true, true, true, true);
        emit Mint(_to, _tokenId);

        // Mint the token.
        vm.prank(address(l2ERC721Bridge));
        L2NFT.safeMint(_to, _tokenId);

        // Token should be owned by the recipient.
        assertEq(L2NFT.ownerOf(_tokenId), _to);
    }

    /// @notice Tests that the `safeMint` function reverts when called by a non-bridge address.
    function testFuzz_safeMint_notBridge_reverts(address _caller) external {
        vm.assume(_caller != address(l2ERC721Bridge));
        vm.expectRevert("OptimismMintableERC721: only bridge can call this function");
        vm.prank(_caller);
        L2NFT.safeMint(alice, 1);
    }
}

/// @title OptimismMintableERC721_Burn_Test
/// @notice Tests the `burn` function of the `OptimismMintableERC721` contract.
contract OptimismMintableERC721_Burn_Test is OptimismMintableERC721_TestInit {
    /// @notice Tests that the `burn` function successfully burns a token when called by the
    ///         bridge.
    function testFuzz_burn_validParams_succeeds(uint256 _tokenId) external {
        // Mint the token first.
        vm.prank(address(l2ERC721Bridge));
        L2NFT.safeMint(alice, _tokenId);

        // Expect a transfer event.
        vm.expectEmit(true, true, true, true);
        emit Transfer(alice, address(0), _tokenId);

        // Expect a burn event.
        vm.expectEmit(true, true, true, true);
        emit Burn(alice, _tokenId);

        // Burn the token.
        vm.prank(address(l2ERC721Bridge));
        L2NFT.burn(alice, _tokenId);

        // Token should no longer exist.
        vm.expectRevert("ERC721: invalid token ID");
        L2NFT.ownerOf(_tokenId);
    }

    /// @notice Tests that the `burn` function reverts when called by a non-bridge address.
    function testFuzz_burn_notBridge_reverts(address _caller) external {
        vm.assume(_caller != address(l2ERC721Bridge));

        // Mint the token first.
        vm.prank(address(l2ERC721Bridge));
        L2NFT.safeMint(alice, 1);

        // Try to burn the token.
        vm.expectRevert("OptimismMintableERC721: only bridge can call this function");
        vm.prank(_caller);
        L2NFT.burn(alice, 1);
    }
}

/// @title OptimismMintableERC721_SupportsInterface_Test
/// @notice Tests the `supportsInterface` function of the `OptimismMintableERC721` contract.
contract OptimismMintableERC721_SupportsInterface_Test is OptimismMintableERC721_TestInit {
    /// @notice Tests that the `supportsInterface` function returns true for
    ///         IOptimismMintableERC721, IERC721Enumerable, IERC721 and IERC165 interfaces.
    function test_supportsInterface_supportedInterfaces_succeeds() external view {
        // Checks if the contract supports the IOptimismMintableERC721 interface.
        assertTrue(L2NFT.supportsInterface(type(IOptimismMintableERC721).interfaceId));
        // Checks if the contract supports the IERC721Enumerable interface.
        assertTrue(L2NFT.supportsInterface(type(IERC721Enumerable).interfaceId));
        // Checks if the contract supports the IERC721 interface.
        assertTrue(L2NFT.supportsInterface(type(IERC721).interfaceId));
        // Checks if the contract supports the IERC165 interface.
        assertTrue(L2NFT.supportsInterface(type(IERC165).interfaceId));
    }

    /// @notice Tests that the `supportsInterface` function returns false for unsupported
    ///         interfaces.
    function testFuzz_supportsInterface_unsupportedInterface_fails(bytes4 _interfaceId) external view {
        vm.assume(_interfaceId != type(IOptimismMintableERC721).interfaceId);
        vm.assume(_interfaceId != type(IERC721Enumerable).interfaceId);
        vm.assume(_interfaceId != type(IERC721).interfaceId);
        vm.assume(_interfaceId != type(IERC721Metadata).interfaceId);
        vm.assume(_interfaceId != type(IERC165).interfaceId);
        assertFalse(L2NFT.supportsInterface(_interfaceId));
    }
}

/// @title OptimismMintableERC721_Version_Test
/// @notice Tests the `version` function of the `OptimismMintableERC721` contract.
contract OptimismMintableERC721_Version_Test is OptimismMintableERC721_TestInit {
    /// @notice Tests that version returns a valid semver string.
    function test_version_validFormat_succeeds() external view {
        SemverComp.parse(L2NFT.version());
    }
}

/// @title OptimismMintableERC721_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the
///         `OptimismMintableERC721` contract.
contract OptimismMintableERC721_Uncategorized_Test is OptimismMintableERC721_TestInit {
    /// @notice Tests that the `tokenURI` function returns the correct URI for a minted token.
    function testFuzz_tokenURI_validTokenId_succeeds(uint256 _tokenId) external {
        // Mint the token first.
        vm.prank(address(l2ERC721Bridge));
        L2NFT.safeMint(alice, _tokenId);

        // Token URI should be correct.
        assertEq(
            L2NFT.tokenURI(_tokenId),
            string(
                abi.encodePacked(
                    "ethereum:",
                    Strings.toHexString(uint160(address(L1NFT)), 20),
                    "@",
                    Strings.toString(1),
                    "/tokenURI?uint256=",
                    Strings.toString(_tokenId)
                )
            )
        );
    }
}
