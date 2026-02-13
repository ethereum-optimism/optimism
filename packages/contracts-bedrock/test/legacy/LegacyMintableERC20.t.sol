// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

import { LegacyMintableERC20 } from "src/legacy/LegacyMintableERC20.sol";
import { ILegacyMintableERC20 } from "interfaces/legacy/ILegacyMintableERC20.sol";

/// @title LegacyMintableERC20_TestInit
/// @notice Reusable test initialization for `LegacyMintableERC20` tests.
abstract contract LegacyMintableERC20_TestInit is CommonTest {
    event Mint(address indexed account, uint256 amount);
    event Burn(address indexed account, uint256 amount);

    LegacyMintableERC20 legacyMintableERC20;

    function setUp() public override {
        super.setUp();

        legacyMintableERC20 = new LegacyMintableERC20(address(l2StandardBridge), address(L1Token), "_L2Token_", "_L2T_");
    }
}

/// @title LegacyMintableERC20_Constructor_Test
/// @notice Tests the constructor of the `LegacyMintableERC20` contract.
contract LegacyMintableERC20_Constructor_Test is LegacyMintableERC20_TestInit {
    /// @notice Tests that the constructor sets the correct values
    function test_constructor_succeeds() public view {
        assertEq(legacyMintableERC20.l2Bridge(), address(l2StandardBridge));
        assertEq(legacyMintableERC20.l1Token(), address(L1Token));
        assertEq(legacyMintableERC20.name(), "_L2Token_");
        assertEq(legacyMintableERC20.symbol(), "_L2T_");
        assertEq(legacyMintableERC20.decimals(), 18);
    }
}

/// @title LegacyMintableERC20_SupportsInterface_Test
/// @notice Tests the `supportsInterface` function of the `LegacyMintableERC20` contract.
contract LegacyMintableERC20_SupportsInterface_Test is LegacyMintableERC20_TestInit {
    /// @notice Tests that the contract supports the correct interfaces
    function test_supportsInterface_supportedInterfaces_succeeds() public view {
        assertEq(legacyMintableERC20.supportsInterface(bytes4(keccak256("supportsInterface(bytes4)"))), true);
        assertEq(
            legacyMintableERC20.supportsInterface(
                ILegacyMintableERC20.l1Token.selector ^ ILegacyMintableERC20.mint.selector
                    ^ ILegacyMintableERC20.burn.selector
            ),
            true
        );
    }

    /// @notice Tests that unsupported interface IDs return false.
    function testFuzz_supportsInterface_unsupportedInterface_succeeds(bytes4 _interfaceId) public view {
        bytes4 erc165 = bytes4(keccak256("supportsInterface(bytes4)"));
        bytes4 legacy = ILegacyMintableERC20.l1Token.selector ^ ILegacyMintableERC20.mint.selector
            ^ ILegacyMintableERC20.burn.selector;
        vm.assume(_interfaceId != erc165 && _interfaceId != legacy);
        assertEq(legacyMintableERC20.supportsInterface(_interfaceId), false);
    }
}

/// @title LegacyMintableERC20_Mint_Test
/// @notice Tests the `mint` function of the `LegacyMintableERC20` contract.
contract LegacyMintableERC20_Mint_Test is LegacyMintableERC20_TestInit {
    /// @notice Tests that mint succeeds when called by the bridge.
    function testFuzz_mint_byBridge_succeeds(address _to, uint256 _amount) public {
        vm.assume(_to != address(0));
        vm.expectEmit(true, false, false, true);
        emit Mint(_to, _amount);
        vm.prank(address(l2StandardBridge));
        legacyMintableERC20.mint(_to, _amount);
        assertEq(legacyMintableERC20.balanceOf(_to), _amount);
    }

    /// @notice Tests that mint reverts for non-bridge callers.
    function testFuzz_mint_byNonBridge_reverts(address _caller, address _to, uint256 _amount) public {
        vm.assume(_caller != address(l2StandardBridge));
        vm.prank(_caller);
        vm.expectRevert(bytes("Only L2 Bridge can mint and burn"));
        legacyMintableERC20.mint(_to, _amount);
    }
}

/// @title LegacyMintableERC20_Burn_Test
/// @notice Tests the `burn` function of the `LegacyMintableERC20` contract.
contract LegacyMintableERC20_Burn_Test is LegacyMintableERC20_TestInit {
    /// @notice Tests that burn succeeds when called by the bridge.
    function testFuzz_burn_byBridge_succeeds(address _from, uint256 _amount) public {
        vm.assume(_from != address(0));
        vm.prank(address(l2StandardBridge));
        legacyMintableERC20.mint(_from, _amount);

        vm.expectEmit(true, false, false, true);
        emit Burn(_from, _amount);
        vm.prank(address(l2StandardBridge));
        legacyMintableERC20.burn(_from, _amount);
        assertEq(legacyMintableERC20.balanceOf(_from), 0);
    }

    /// @notice Tests that burn reverts for non-bridge callers.
    function testFuzz_burn_byNonBridge_reverts(address _caller, address _from, uint256 _amount) public {
        vm.assume(_caller != address(l2StandardBridge));
        vm.prank(_caller);
        vm.expectRevert(bytes("Only L2 Bridge can mint and burn"));
        legacyMintableERC20.burn(_from, _amount);
    }
}
