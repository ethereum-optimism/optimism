// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { IOptimismMintableERC20 } from "interfaces/universal/IOptimismMintableERC20.sol";
import { ILegacyMintableERC20 } from "interfaces/legacy/ILegacyMintableERC20.sol";
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

/// @title OptimismMintableERC20_TestInit
/// @notice Reusable test initialization for `OptimismMintableERC20` tests.
abstract contract OptimismMintableERC20_TestInit is CommonTest {
    event Mint(address indexed account, uint256 amount);
    event Burn(address indexed account, uint256 amount);
}

/// @title OptimismMintableERC20_Permit2_Test
/// @notice Tests the `PERMIT2` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_Permit2_Test is OptimismMintableERC20_TestInit {
    /// @notice Tests that PERMIT2 can transfer tokens on behalf of any owner.
    function testFuzz_permit2_transferFrom_succeeds(address _owner, address _to, uint256 _amount) external {
        vm.assume(_owner != address(0));
        vm.assume(_to != address(0));
        vm.assume(_owner != _to);

        vm.prank(address(l2StandardBridge));
        L2Token.mint(_owner, _amount);

        assertEq(L2Token.balanceOf(_to), 0);
        vm.prank(L2Token.PERMIT2());
        L2Token.transferFrom(_owner, _to, _amount);
        assertEq(L2Token.balanceOf(_to), _amount);
    }
}

/// @title OptimismMintableERC20_Allowance_Test
/// @notice Tests the `allowance` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_Allowance_Test is OptimismMintableERC20_TestInit {
    /// @notice Tests that allowance returns max uint256 for PERMIT2 spender.
    function testFuzz_allowance_permit2Spender_succeeds(address _owner) external view {
        assertEq(L2Token.allowance(_owner, L2Token.PERMIT2()), type(uint256).max);
    }

    /// @notice Tests that allowance returns the actual allowance for non-PERMIT2 spenders.
    function testFuzz_allowance_nonPermit2Spender_succeeds(address _owner, address _spender) external view {
        vm.assume(_spender != L2Token.PERMIT2());
        assertEq(L2Token.allowance(_owner, _spender), 0);
    }
}

/// @title OptimismMintableERC20_Mint_Test
/// @notice Tests the `mint` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_Mint_Test is OptimismMintableERC20_TestInit {
    /// @notice Tests that minting tokens succeeds when called by the bridge.
    function testFuzz_mint_fromBridge_succeeds(address _to, uint256 _amount) external {
        vm.assume(_to != address(0));

        vm.expectEmit(true, true, true, true);
        emit Mint(_to, _amount);

        vm.prank(address(l2StandardBridge));
        L2Token.mint(_to, _amount);

        assertEq(L2Token.balanceOf(_to), _amount);
    }

    /// @notice Tests that minting reverts when called by a non-bridge address.
    function testFuzz_mint_notBridge_reverts(address _caller) external {
        vm.assume(_caller != address(l2StandardBridge));

        vm.expectRevert("OptimismMintableERC20: only bridge can mint and burn");
        vm.prank(_caller);
        L2Token.mint(alice, 100);
    }
}

/// @title OptimismMintableERC20_Burn_Test
/// @notice Tests the `burn` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_Burn_Test is OptimismMintableERC20_TestInit {
    /// @notice Tests that burning tokens succeeds when called by the bridge.
    function testFuzz_burn_fromBridge_succeeds(address _from, uint256 _amount) external {
        vm.assume(_from != address(0));

        vm.prank(address(l2StandardBridge));
        L2Token.mint(_from, _amount);

        vm.expectEmit(true, true, true, true);
        emit Burn(_from, _amount);

        vm.prank(address(l2StandardBridge));
        L2Token.burn(_from, _amount);

        assertEq(L2Token.balanceOf(_from), 0);
    }

    /// @notice Tests that burning reverts when called by a non-bridge address.
    function testFuzz_burn_notBridge_reverts(address _caller) external {
        vm.assume(_caller != address(l2StandardBridge));

        vm.expectRevert("OptimismMintableERC20: only bridge can mint and burn");
        vm.prank(_caller);
        L2Token.burn(alice, 100);
    }
}

/// @title OptimismMintableERC20_SupportsInterface_Test
/// @notice Tests the `supportsInterface` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_SupportsInterface_Test is OptimismMintableERC20_TestInit {
    /// @notice Tests that the contract supports ERC165, ILegacyMintableERC20, and
    ///         IOptimismMintableERC20 interfaces.
    function test_supportsInterface_supportedInterfaces_succeeds() external view {
        // The assertEq calls in this test are comparing the manual calculation of the iface, with
        // what is returned by the solidity's type().interfaceId, just to be safe.
        bytes4 iface1 = bytes4(keccak256("supportsInterface(bytes4)"));
        assertEq(iface1, type(IERC165).interfaceId);
        assertTrue(L2Token.supportsInterface(iface1));

        bytes4 iface2 = L2Token.l1Token.selector ^ L2Token.mint.selector ^ L2Token.burn.selector;
        assertEq(iface2, type(ILegacyMintableERC20).interfaceId);
        assertTrue(L2Token.supportsInterface(iface2));

        bytes4 iface3 =
            L2Token.remoteToken.selector ^ L2Token.bridge.selector ^ L2Token.mint.selector ^ L2Token.burn.selector;
        assertEq(iface3, type(IOptimismMintableERC20).interfaceId);
        assertTrue(L2Token.supportsInterface(iface3));
    }

    /// @notice Tests that the contract returns false for unsupported interfaces.
    function testFuzz_supportsInterface_unsupportedInterface_succeeds(bytes4 _interfaceId) external view {
        vm.assume(_interfaceId != type(IERC165).interfaceId);
        vm.assume(_interfaceId != type(ILegacyMintableERC20).interfaceId);
        vm.assume(_interfaceId != type(IOptimismMintableERC20).interfaceId);

        assertFalse(L2Token.supportsInterface(_interfaceId));
    }
}

/// @title OptimismMintableERC20_L1Token_Test
/// @notice Tests the `l1Token` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_L1Token_Test is OptimismMintableERC20_TestInit {
    function test_l1Token_succeeds() external view {
        assertEq(L2Token.l1Token(), address(L1Token));
    }
}

/// @title OptimismMintableERC20_L2Bridge_Test
/// @notice Tests the `l2Bridge` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_L2Bridge_Test is OptimismMintableERC20_TestInit {
    function test_l2Bridge_succeeds() external view {
        assertEq(L2Token.l2Bridge(), address(l2StandardBridge));
    }
}

/// @title OptimismMintableERC20_RemoteToken_Test
/// @notice Tests the `remoteToken` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_RemoteToken_Test is OptimismMintableERC20_TestInit {
    function test_remoteToken_succeeds() external view {
        assertEq(L2Token.remoteToken(), address(L1Token));
    }
}

/// @title OptimismMintableERC20_Bridge_Test
/// @notice Tests the `bridge` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_Bridge_Test is OptimismMintableERC20_TestInit {
    function test_bridge_succeeds() external view {
        assertEq(L2Token.bridge(), address(l2StandardBridge));
    }
}

/// @title OptimismMintableERC20_Decimals_Test
/// @notice Tests the `decimals` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_Decimals_Test is OptimismMintableERC20_TestInit {
    /// @notice Tests that decimals returns the expected value.
    function test_decimals_succeeds() external view {
        assertEq(L2Token.decimals(), 18);
    }
}

/// @title OptimismMintableERC20_Version_Test
/// @notice Tests the `version` function of the `OptimismMintableERC20` contract.
contract OptimismMintableERC20_Version_Test is OptimismMintableERC20_TestInit {
    /// @notice Tests that version returns a valid semver string.
    function test_version_validFormat_succeeds() external view {
        SemverComp.parse(L2Token.version());
    }
}

/// @title OptimismMintableERC20_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `OptimismMintableERC20`
///         contract.
contract OptimismMintableERC20_Uncategorized_Test is OptimismMintableERC20_TestInit {
    /// @notice Tests that legacy getters return the expected values.
    function test_legacy_succeeds() external view {
        // Getters for the remote token
        assertEq(L2Token.REMOTE_TOKEN(), address(L1Token));
        assertEq(L2Token.remoteToken(), address(L1Token));
        assertEq(L2Token.l1Token(), address(L1Token));
        // Getters for the bridge
        assertEq(L2Token.BRIDGE(), address(l2StandardBridge));
        assertEq(L2Token.bridge(), address(l2StandardBridge));
        assertEq(L2Token.l2Bridge(), address(l2StandardBridge));
    }
}
