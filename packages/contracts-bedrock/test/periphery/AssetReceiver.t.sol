// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { TestERC20 } from "test/mocks/TestERC20.sol";
import { TestERC721 } from "test/mocks/TestERC721.sol";

// Contracts
import { AssetReceiver } from "src/periphery/AssetReceiver.sol";

/// @title AssetReceiver_TestInit
/// @notice Reusable test initialization for `AssetReceiver` tests.
abstract contract AssetReceiver_TestInit is Test {
    address alice = address(128);
    address bob = address(256);

    uint8 immutable DEFAULT_TOKEN_ID = 0;

    TestERC20 testERC20;
    TestERC721 testERC721;
    AssetReceiver assetReceiver;

    event ReceivedETH(address indexed from, uint256 amount);
    event WithdrewETH(address indexed withdrawer, address indexed recipient, uint256 amount);
    event WithdrewERC20(address indexed withdrawer, address indexed recipient, address indexed asset, uint256 amount);
    event WithdrewERC721(address indexed withdrawer, address indexed recipient, address indexed asset, uint256 id);

    function setUp() public {
        // Deploy ERC20 and ERC721 tokens
        testERC20 = new TestERC20();
        testERC721 = new TestERC721();

        // Deploy AssetReceiver contract
        assetReceiver = new AssetReceiver(address(alice));
        vm.label(address(assetReceiver), "AssetReceiver");

        // Give alice and bob some ETH
        vm.deal(alice, 1 ether);
        vm.deal(bob, 1 ether);

        testERC721.mint(alice, DEFAULT_TOKEN_ID);

        vm.label(alice, "alice");
        vm.label(bob, "bob");
    }
}

/// @title AssetReceiver_Constructor_Test
/// @notice Tests the constructor of the `AssetReceiver` contract.
contract AssetReceiver_Constructor_Test is AssetReceiver_TestInit {
    /// @notice Tests if the owner was set correctly during deploy.
    function test_constructor_succeeds() external view {
        assertEq(address(alice), assetReceiver.owner());
    }
}

/// @title AssetReceiver_Receive_Test
/// @notice Tests the `receive` function of the `AssetReceiver` contract.
contract AssetReceiver_Receive_Test is AssetReceiver_TestInit {
    /// @notice Tests that receive works as intended.
    function test_receive_succeeds() external {
        // Check that contract balance is 0 initially
        assertEq(address(assetReceiver).balance, 0);

        vm.expectEmit(address(assetReceiver));
        emit ReceivedETH(alice, 100);
        // Send funds
        vm.prank(alice);
        (bool success,) = address(assetReceiver).call{ value: 100 }(hex"");

        // Compare balance after the tx sent
        assertTrue(success);
        assertEq(address(assetReceiver).balance, 100);
    }
}

/// @title AssetReceiver_WithdrawETH_Test
/// @notice Tests the `withdrawETH` function of the `AssetReceiver` contract.
contract AssetReceiver_WithdrawETH_Test is AssetReceiver_TestInit {
    /// @notice Tests withdrawETH function with only an address as an argument, called by owner.
    function test_withdrawETH_succeeds() external {
        // Check contract initial balance
        assertEq(address(assetReceiver).balance, 0);
        // Fund contract with 1 eth and check caller and contract balances
        vm.deal(address(assetReceiver), 1 ether);
        assertEq(address(assetReceiver).balance, 1 ether);

        assertEq(address(alice).balance, 1 ether);

        vm.expectEmit(address(assetReceiver));
        emit WithdrewETH(alice, alice, 1 ether);

        // Call withdrawETH
        vm.prank(alice);
        assetReceiver.withdrawETH(payable(alice));

        // Check balances after the call
        assertEq(address(assetReceiver).balance, 0);
        assertEq(address(alice).balance, 2 ether);
    }

    /// @notice withdrawETH should fail if called by non-owner.
    function test_withdrawETH_unauthorized_reverts() external {
        vm.deal(address(assetReceiver), 1 ether);
        vm.expectRevert("UNAUTHORIZED");
        assetReceiver.withdrawETH(payable(alice));
    }

    /// @notice Similar as withdrawETH but specify amount to withdraw.
    function test_withdrawETH_withAmount_succeeds() external {
        assertEq(address(assetReceiver).balance, 0);

        vm.deal(address(assetReceiver), 1 ether);
        assertEq(address(assetReceiver).balance, 1 ether);

        assertEq(address(alice).balance, 1 ether);

        vm.expectEmit(address(assetReceiver));
        emit WithdrewETH(alice, alice, 0.5 ether);

        // Call withdrawETH
        vm.prank(alice);
        assetReceiver.withdrawETH(payable(alice), 0.5 ether);

        // Check balances after the call
        assertEq(address(assetReceiver).balance, 0.5 ether);
        assertEq(address(alice).balance, 1.5 ether);
    }

    /// @notice withdrawETH with address and amount as arguments called by non-owner.
    function test_withdrawETH_withAmountUnauthorized_reverts() external {
        vm.deal(address(assetReceiver), 1 ether);
        vm.expectRevert("UNAUTHORIZED");
        assetReceiver.withdrawETH(payable(alice), 0.5 ether);
    }
}

/// @title AssetReceiver_WithdrawERC20_Test
/// @notice Tests the `withdrawERC20` function of the `AssetReceiver` contract.
contract AssetReceiver_WithdrawERC20_Test is AssetReceiver_TestInit {
    /// @notice Test withdrawERC20 with token and address arguments, from owner.
    function test_withdrawERC20_succeeds() external {
        // Check balances before the call
        assertEq(testERC20.balanceOf(address(assetReceiver)), 0);

        deal(address(testERC20), address(assetReceiver), 100_000);
        assertEq(testERC20.balanceOf(address(assetReceiver)), 100_000);
        assertEq(testERC20.balanceOf(alice), 0);

        vm.expectEmit(address(assetReceiver));
        emit WithdrewERC20(alice, alice, address(testERC20), 100_000);

        // Call withdrawERC20
        vm.prank(alice);
        assetReceiver.withdrawERC20(testERC20, alice);

        // Check balances after the call
        assertEq(testERC20.balanceOf(alice), 100_000);
        assertEq(testERC20.balanceOf(address(assetReceiver)), 0);
    }

    /// @notice Same as withdrawERC20 but call from non-owner.
    function test_withdrawERC20_unauthorized_reverts() external {
        deal(address(testERC20), address(assetReceiver), 100_000);
        vm.expectRevert("UNAUTHORIZED");
        assetReceiver.withdrawERC20(testERC20, alice);
    }

    /// @notice Similar as withdrawERC20 but specify amount to withdraw.
    function test_withdrawERC20_withAmount_succeeds() external {
        // Check balances before the call
        assertEq(testERC20.balanceOf(address(assetReceiver)), 0);

        deal(address(testERC20), address(assetReceiver), 100_000);
        assertEq(testERC20.balanceOf(address(assetReceiver)), 100_000);
        assertEq(testERC20.balanceOf(alice), 0);

        vm.expectEmit(address(assetReceiver));
        emit WithdrewERC20(alice, alice, address(testERC20), 50_000);

        // Call withdrawERC20
        vm.prank(alice);
        assetReceiver.withdrawERC20(testERC20, alice, 50_000);

        // check balances after the call
        assertEq(testERC20.balanceOf(alice), 50_000);
        assertEq(testERC20.balanceOf(address(assetReceiver)), 50_000);
    }

    /// @notice Similar as withdrawERC20 with amount but call from non-owner.
    function test_withdrawERC20_withAmountUnauthorized_reverts() external {
        deal(address(testERC20), address(assetReceiver), 100_000);
        vm.expectRevert("UNAUTHORIZED");
        assetReceiver.withdrawERC20(testERC20, alice, 50_000);
    }
}

/// @title AssetReceiver_WithdrawERC721_Test
/// @notice Tests the `withdrawERC721` function of the `AssetReceiver` contract.
contract AssetReceiver_WithdrawERC721_Test is AssetReceiver_TestInit {
    /// @notice Test withdrawERC721 from owner.
    function test_withdrawERC721_succeeds() external {
        // Check owner of the token before calling withdrawERC721
        assertEq(testERC721.ownerOf(DEFAULT_TOKEN_ID), alice);

        // Send the token from alice to the contract
        vm.prank(alice);
        testERC721.transferFrom(alice, address(assetReceiver), DEFAULT_TOKEN_ID);
        assertEq(testERC721.ownerOf(DEFAULT_TOKEN_ID), address(assetReceiver));

        vm.expectEmit(address(assetReceiver));
        emit WithdrewERC721(alice, alice, address(testERC721), DEFAULT_TOKEN_ID);

        // Call withdrawERC721
        vm.prank(alice);
        assetReceiver.withdrawERC721(testERC721, alice, DEFAULT_TOKEN_ID);

        // Check the owner after the call
        assertEq(testERC721.ownerOf(DEFAULT_TOKEN_ID), alice);
    }

    /// @notice Similar as withdrawERC721 but call from non-owner.
    function test_withdrawERC721_unauthorized_reverts() external {
        vm.prank(alice);
        testERC721.transferFrom(alice, address(assetReceiver), DEFAULT_TOKEN_ID);
        vm.expectRevert("UNAUTHORIZED");
        assetReceiver.withdrawERC721(testERC721, alice, DEFAULT_TOKEN_ID);
    }
}
