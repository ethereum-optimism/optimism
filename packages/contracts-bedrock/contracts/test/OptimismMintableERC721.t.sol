// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ERC721, IERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import {
    IERC721Enumerable
} from "@openzeppelin/contracts/token/ERC721/extensions/ERC721Enumerable.sol";
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { ERC721Bridge_Initializer } from "./CommonTest.t.sol";
import {
    OptimismMintableERC721,
    IOptimismMintableERC721
} from "../universal/OptimismMintableERC721.sol";

contract OptimismMintableERC721_Test is ERC721Bridge_Initializer {
    ERC721 internal L1Token;
    OptimismMintableERC721 internal L2Token;

    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);

    event Mint(address indexed account, uint256 tokenId);

    event Burn(address indexed account, uint256 tokenId);

    function setUp() public override {
        super.setUp();

        // Set up the token pair.
        L1Token = new ERC721("L1Token", "L1T");
        L2Token = new OptimismMintableERC721(
            address(L2Bridge),
            1,
            address(L1Token),
            "L2Token",
            "L2T"
        );

        // Label the addresses for nice traces.
        vm.label(address(L1Token), "L1ERC721Token");
        vm.label(address(L2Token), "L2ERC721Token");
    }

    function test_constructor_succeeds() external {
        assertEq(L2Token.name(), "L2Token");
        assertEq(L2Token.symbol(), "L2T");
        assertEq(L2Token.remoteToken(), address(L1Token));
        assertEq(L2Token.bridge(), address(L2Bridge));
        assertEq(L2Token.remoteChainId(), 1);
        assertEq(L2Token.REMOTE_TOKEN(), address(L1Token));
        assertEq(L2Token.BRIDGE(), address(L2Bridge));
        assertEq(L2Token.REMOTE_CHAIN_ID(), 1);
    }

    /// @notice Ensure that the contract supports the expected interfaces.
    function test_supportsInterfaces_succeeds() external {
        // Checks if the contract supports the IOptimismMintableERC721 interface.
        assertTrue(L2Token.supportsInterface(type(IOptimismMintableERC721).interfaceId));
        // Checks if the contract supports the IERC721Enumerable interface.
        assertTrue(L2Token.supportsInterface(type(IERC721Enumerable).interfaceId));
        // Checks if the contract supports the IERC721 interface.
        assertTrue(L2Token.supportsInterface(type(IERC721).interfaceId));
        // Checks if the contract supports the IERC165 interface.
        assertTrue(L2Token.supportsInterface(type(IERC165).interfaceId));
    }

    function test_safeMint_succeeds() external {
        // Expect a transfer event.
        vm.expectEmit(true, true, true, true);
        emit Transfer(address(0), alice, 1);

        // Expect a mint event.
        vm.expectEmit(true, true, true, true);
        emit Mint(alice, 1);

        // Mint the token.
        vm.prank(address(L2Bridge));
        L2Token.safeMint(alice, 1);

        // Token should be owned by alice.
        assertEq(L2Token.ownerOf(1), alice);
    }

    function test_safeMint_notBridge_reverts() external {
        // Try to mint the token.
        vm.expectRevert("OptimismMintableERC721: only bridge can call this function");
        vm.prank(address(alice));
        L2Token.safeMint(alice, 1);
    }

    function test_burn_succeeds() external {
        // Mint the token first.
        vm.prank(address(L2Bridge));
        L2Token.safeMint(alice, 1);

        // Expect a transfer event.
        vm.expectEmit(true, true, true, true);
        emit Transfer(alice, address(0), 1);

        // Expect a burn event.
        vm.expectEmit(true, true, true, true);
        emit Burn(alice, 1);

        // Burn the token.
        vm.prank(address(L2Bridge));
        L2Token.burn(alice, 1);

        // Token should be owned by address(0).
        vm.expectRevert("ERC721: invalid token ID");
        L2Token.ownerOf(1);
    }

    function test_burn_notBridge_reverts() external {
        // Mint the token first.
        vm.prank(address(L2Bridge));
        L2Token.safeMint(alice, 1);

        // Try to burn the token.
        vm.expectRevert("OptimismMintableERC721: only bridge can call this function");
        vm.prank(address(alice));
        L2Token.burn(alice, 1);
    }

    function test_tokenURI_succeeds() external {
        // Mint the token first.
        vm.prank(address(L2Bridge));
        L2Token.safeMint(alice, 1);

        // Token URI should be correct.
        assertEq(
            L2Token.tokenURI(1),
            string(
                abi.encodePacked(
                    "ethereum:",
                    Strings.toHexString(uint160(address(L1Token)), 20),
                    "@",
                    Strings.toString(1),
                    "/tokenURI?uint256=",
                    Strings.toString(1)
                )
            )
        );
    }
}
