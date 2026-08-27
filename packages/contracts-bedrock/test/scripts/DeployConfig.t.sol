// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { DeployConfig } from "scripts/deploy/DeployConfig.s.sol";

/// @title DeployConfigTest
/// @notice Tests legacy configuration compatibility checks in DeployConfig.
contract DeployConfigTest is Test {
    DeployConfig deployConfig;

    function setUp() public {
        deployConfig = new DeployConfig();
    }

    /// @notice Ensures retired Alt-DA deployments fail instead of silently using Ethereum DA.
    function test_read_useAltDATrue_reverts() public {
        string memory path = ".testdata/DeployConfig.useAltDA.json";
        vm.createDir(".testdata", true);
        vm.writeFile(path, '{"useAltDA":true}');

        vm.expectRevert(DeployConfig.AltDANoLongerSupported.selector);
        deployConfig.read(path);

        vm.removeFile(path);
    }
}
