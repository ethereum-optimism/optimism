// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Blueprint } from "src/libraries/Blueprint.sol";

contract Blueprint_Test is Test {
    // TODO add tests that things revert if an address has no code.

    function test_roundtrip_succeeds(bytes memory _initcode) public {
        vm.assume(_initcode.length > 0);

        // Convert the initcode to match the ERC-5202 blueprint format.
        bytes memory blueprintInitcode = Blueprint.blueprintDeployerBytecode(_initcode);

        // Deploy the blueprint.
        address blueprintAddress;
        assembly ("memory-safe") {
            blueprintAddress := create2(0, add(blueprintInitcode, 0x20), mload(blueprintInitcode), 0)
        }
        require(blueprintAddress != address(0), "DeployImplementations: create2 failed");

        // Read the blueprint code from the deployed code.
        bytes memory blueprintCode = address(blueprintAddress).code;

        // Parse the blueprint preamble.
        Blueprint.Preamble memory preamble = Blueprint.parseBlueprintPreamble(blueprintCode);
        assertEq(preamble.ercVersion, 0, "100");
        assertEq(preamble.preambleData, hex"", "200");
        assertEq(preamble.initcode, _initcode, "300");
    }

    function test_bytesToUint_succeeds() public pure {
        // These test cases (and the logic for bytesToUint) are taken from forge-std.
        assertEq(3, Blueprint.bytesToUint(hex"03"));
        assertEq(2, Blueprint.bytesToUint(hex"02"));
        assertEq(255, Blueprint.bytesToUint(hex"ff"));
        assertEq(29625, Blueprint.bytesToUint(hex"73b9"));

        // Additional test cases.
        assertEq(0, Blueprint.bytesToUint(hex""));
        assertEq(0, Blueprint.bytesToUint(hex"00"));
        assertEq(14545064521499334880, Blueprint.bytesToUint(hex"c9da731e871ad8e0"));
        assertEq(type(uint256).max, Blueprint.bytesToUint(bytes.concat(bytes32(type(uint256).max))));
    }
}
