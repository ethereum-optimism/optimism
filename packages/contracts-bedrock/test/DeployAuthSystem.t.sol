// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { stdToml } from "forge-std/StdToml.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

import { DeployAuthSystemInput, DeployAuthSystemOutput } from "scripts/DeployAuthSystem.s.sol";

contract DeployAuthSystemInput_Test is Test {
    DeployAuthSystemInput dsi;

    uint256 threshold = 5;
    address[] owners;

    function setUp() public {
        dsi = new DeployAuthSystemInput();
        address[] memory _owners = Solarray.addresses(
            0x1111111111111111111111111111111111111111,
            0x2222222222222222222222222222222222222222,
            0x3333333333333333333333333333333333333333,
            0x4444444444444444444444444444444444444444,
            0x5555555555555555555555555555555555555555,
            0x6666666666666666666666666666666666666666,
            0x7777777777777777777777777777777777777777
        );

        for (uint256 i = 0; i < _owners.length; i++) {
            owners.push(_owners[i]);
        }
    }

    function test_loadInputFile_succeeds() public {
        string memory root = vm.projectRoot();
        string memory path = string.concat(root, "/test/fixtures/test-deploy-auth-system-in.toml");

        dsi.loadInputFile(path);

        assertEq(threshold, dsi.threshold(), "100");
        assertEq(owners.length, dsi.owners().length, "200");
    }

    function test_getters_whenNotSet_revert() public {
        vm.expectRevert("DeployAuthSystemInput: threshold not set");
        dsi.threshold();

        vm.expectRevert("DeployAuthSystemInput: owners not set");
        dsi.owners();
    }
}

