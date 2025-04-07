// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, stdStorage, StdStorage } from "forge-std/Test.sol";
import { stdToml } from "forge-std/StdToml.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

import { DeployAuthSystem2 } from "scripts/deploy/DeployAuthSystem2.s.sol";

// contract DeployAuthSystemInput_Test is Test {
//     DeployAuthSystemInput dasi;

//     uint256 threshold = 5;
//     address[] owners;

//     function setUp() public {
//         dasi = new DeployAuthSystemInput();
//         address[] memory _owners = Solarray.addresses(
//             0x1111111111111111111111111111111111111111,
//             0x2222222222222222222222222222222222222222,
//             0x3333333333333333333333333333333333333333,
//             0x4444444444444444444444444444444444444444,
//             0x5555555555555555555555555555555555555555,
//             0x6666666666666666666666666666666666666666,
//             0x7777777777777777777777777777777777777777
//         );

//         for (uint256 i = 0; i < _owners.length; i++) {
//             owners.push(_owners[i]);
//         }
//     }

//     function test_getters_whenNotSet_reverts() public {
//         vm.expectRevert("DeployAuthSystemInput: threshold not set");
//         dasi.threshold();

//         vm.expectRevert("DeployAuthSystemInput: owners not set");
//         dasi.owners();
//     }

//     function test_setters_ownerAlreadySet_reverts() public {
//         dasi.set(dasi.owners.selector, owners);

//         vm.expectRevert("DeployAuthSystemInput: owners already set");
//         dasi.set(dasi.owners.selector, owners);
//     }
// }

// contract DeployAuthSystemOutput_Test is Test {
//     using stdToml for string;

//     DeployAuthSystemOutput daso;

//     function setUp() public {
//         daso = new DeployAuthSystemOutput();
//     }

//     function test_set_succeeds() public {
//         address safeAddr = makeAddr("safe");

//         vm.etch(safeAddr, hex"01");

//         daso.set(daso.safe.selector, safeAddr);

//         assertEq(safeAddr, address(daso.safe()), "100");
//     }

//     function test_getter_whenNotSet_reverts() public {
//         vm.expectRevert("DeployUtils: zero address");
//         daso.safe();
//     }

//     function test_getter_whenAddrHasNoCode_reverts() public {
//         address emptyAddr = makeAddr("emptyAddr");
//         bytes memory expectedErr = bytes(string.concat("DeployUtils: no code at ", vm.toString(emptyAddr)));

//         daso.set(daso.safe.selector, emptyAddr);
//         vm.expectRevert(expectedErr);
//         daso.safe();
//     }
// }

contract DeployAuthSystem2_Test is Test {
    using stdStorage for StdStorage;

    DeployAuthSystem2 deployAuthSystem;

    // Define default input variables for testing.
    uint256 defaultThreshold = 5;
    uint256 defaultOwnersLength = 7;
    address[] defaultOwners;

    function setUp() public {
        deployAuthSystem = new DeployAuthSystem2();

        for (uint256 i = 0; i < defaultOwnersLength; i++) {
            defaultOwners.push(makeAddr(string.concat("owner", vm.toString(i))));
        }
    }

    function hash(bytes32 _seed, uint256 _i) internal pure returns (bytes32) {
        return keccak256(abi.encode(_seed, _i));
    }

    function testFuzz_run_succeeds(bytes32 _seed, uint8 _numOwners, uint64 _threshold) public {
        vm.assume(_threshold > 0);
        vm.assume(_numOwners >= _threshold);

        address[] memory owners = new address[](_numOwners);
        for (uint8 i = 0; i < _numOwners; i++) {
            owners[i] = address(uint160(uint256(hash(_seed, i))));
        }

        DeployAuthSystem2.Input memory input = DeployAuthSystem2.Input(_threshold, owners);

        DeployAuthSystem2.Output memory output = deployAuthSystem.run(input);

        assertNotEq(address(output.safe), address(0), "100");
        assertEq(output.safe.getThreshold(), _threshold, "200");

        // TODO The rest of the Safe setup is not finished atm
    }

    function test_run_nullInput_reverts() public {
        DeployAuthSystem2.Input memory input;

        input = DeployAuthSystem2.Input(0, Solarray.addresses(0x1111111111111111111111111111111111111111));
        vm.expectRevert("DeployAuthSystem: threshold not set");
        deployAuthSystem.run(input);

        input = DeployAuthSystem2.Input(1, Solarray.addresses(address(0)));
        vm.expectRevert("DeployAuthSystem: owner not set");
        deployAuthSystem.run(input);

        input = DeployAuthSystem2.Input(1, new address[](0));
        vm.expectRevert("DeployAuthSystem: owners not set");
        deployAuthSystem.run(input);
    }

    function test_run_thresholdTooLarge_reverts(uint8 _numOwners, uint64 _threshold) public {
        vm.assume(_numOwners != 0);
        vm.assume(_numOwners < _threshold);

        address[] memory owners = new address[](_numOwners);

        DeployAuthSystem2.Input memory input = DeployAuthSystem2.Input(_threshold, owners);
        vm.expectRevert("DeployAuthSystem: threshold too large");
        deployAuthSystem.run(input);
    }
}
