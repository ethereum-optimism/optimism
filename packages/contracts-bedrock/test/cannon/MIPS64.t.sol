// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { UnsupportedStateVersion } from "src/cannon/libraries/CannonErrors.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS2 } from "interfaces/cannon/IMIPS2.sol";

contract MIPS64_Test is Test {
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

    /// @notice Test the we can deploy MIPS64 with a valid version parameter.
    function test_deploy_supportedVersions_succeeds() external {
        for (uint256 i = 0; i < validVersions.length; i++) {
            uint256 version = validVersions[i];
            IMIPS2 mips = deployVm(version);
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

    /// @notice Deploys new MIPS64 contract with the given version parameter.
    function deployVm(uint256 version) internal returns (IMIPS2) {
        return IMIPS2(
            DeployUtils.create1({
                _name: "MIPS64",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IMIPS2.__constructor__, (oracle, version)))
            })
        );
    }
}
