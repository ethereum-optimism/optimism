// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { UnsupportedStateVersion } from "src/cannon/libraries/CannonErrors.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";

/// @title MIPS64_TestInit
/// @notice Reusable test initialization for `MIPS64` tests.
abstract contract MIPS64_TestInit is Test {
    IPreimageOracle oracle;

    // Store some data about acceptable versions
    uint256[2] validVersions = [7, 8];
    mapping(uint256 => bool) public isValidVersion;
    uint256 maxValidVersion;

    /// @notice Sets up the testing suite.
    function setUp() public virtual {
        oracle = IPreimageOracle(
            DeployUtils.create1({
                _name: "PreimageOracle",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IPreimageOracle.__constructor__, (0, 0)))
            })
        );
        vm.label(address(oracle), "PreimageOracle");

        // Store some metadata about versions
        for (uint256 i = 0; i < validVersions.length; i++) {
            uint256 validVersion = validVersions[i];
            isValidVersion[validVersion] = true;
            if (validVersion > maxValidVersion) {
                maxValidVersion = validVersion;
            }
        }
    }

    /// @notice Deploys new MIPS64 contract with the given version parameter.
    function deployVm(uint256 version) internal returns (IMIPS64) {
        return IMIPS64(
            DeployUtils.create1({
                _name: "MIPS64",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IMIPS64.__constructor__, (oracle, version)))
            })
        );
    }
}

/// @title MIPS64_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `MIPS64` contract or
///         are testing multiple functions at once.
contract MIPS64_Uncategorized_Test is MIPS64_TestInit {
    /// @notice Test the we can deploy MIPS64 with a valid version parameter.
    function test_deploy_supportedVersions_succeeds() external {
        for (uint256 i = 0; i < validVersions.length; i++) {
            uint256 version = validVersions[i];
            IMIPS64 mips = deployVm(version);
            assertNotEq(address(mips), address(0));
        }
    }

    /// @notice Test that deploying MIPS64 with an invalid version reverts with expected error.
    function test_deploy_unsupportedVersions_fails() external {
        for (uint256 ver = 0; ver <= maxValidVersion + 2; ver++) {
            if (isValidVersion[ver]) {
                continue;
            }

            vm.expectRevert(abi.encodeWithSelector(UnsupportedStateVersion.selector));
            deployVm(ver);
        }
    }
}
