//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { AttestationStation } from "src/periphery/op-nft/AttestationStation.sol";

contract AttestationStation_Initializer is Test {
    address alice_attestor = address(128);
    address bob = address(256);
    address sally = address(512);

    function setUp() public {
        // Give alice and bob some ETH
        vm.deal(alice_attestor, 1 ether);

        vm.label(alice_attestor, "alice_attestor");
        vm.label(bob, "bob");
        vm.label(sally, "sally");
    }
}

contract AttestationStationTest is AttestationStation_Initializer {
    event AttestationCreated(address indexed creator, address indexed about, bytes32 indexed key, bytes val);

    function test_attest_individual_succeeds() external {
        AttestationStation attestationStation = new AttestationStation();

        vm.expectEmit(true, true, true, true);
        emit AttestationCreated(alice_attestor, bob, bytes32("foo"), bytes("bar"));

        vm.prank(alice_attestor);
        attestationStation.attest({ _about: bob, _key: bytes32("foo"), _val: bytes("bar") });
    }

    function test_attest_single_succeeds() external {
        AttestationStation attestationStation = new AttestationStation();

        AttestationStation.AttestationData[] memory attestationDataArr = new AttestationStation.AttestationData[](1);

        // alice is going to attest about bob
        AttestationStation.AttestationData memory attestationData = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("test-key:string"),
            val: bytes("test-value")
        });

        // assert the attestation starts empty
        assertEq(attestationStation.attestations(alice_attestor, attestationData.about, attestationData.key), "");

        // make attestation
        vm.prank(alice_attestor);
        attestationDataArr[0] = attestationData;
        attestationStation.attest(attestationDataArr);

        // assert the attestation is there
        assertEq(
            attestationStation.attestations(alice_attestor, attestationData.about, attestationData.key),
            attestationData.val
        );

        bytes memory new_val = bytes("new updated value");
        // make a new attestations to same about and key
        attestationData =
            AttestationStation.AttestationData({ about: attestationData.about, key: attestationData.key, val: new_val });

        vm.prank(alice_attestor);
        attestationDataArr[0] = attestationData;
        attestationStation.attest(attestationDataArr);

        // assert the attestation is updated
        assertEq(
            attestationStation.attestations(alice_attestor, attestationData.about, attestationData.key),
            attestationData.val
        );
    }

    function test_attest_bulk_succeeds() external {
        AttestationStation attestationStation = new AttestationStation();

        vm.prank(alice_attestor);

        AttestationStation.AttestationData[] memory attestationData = new AttestationStation.AttestationData[](3);
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("test-key:string"),
            val: bytes("test-value")
        });

        attestationData[1] =
            AttestationStation.AttestationData({ about: bob, key: bytes32("test-key2"), val: bytes("test-value2") });

        attestationData[2] = AttestationStation.AttestationData({
            about: sally,
            key: bytes32("test-key:string"),
            val: bytes("test-value3")
        });

        attestationStation.attest(attestationData);

        // assert the attestations are there
        assertEq(
            attestationStation.attestations(alice_attestor, attestationData[0].about, attestationData[0].key),
            attestationData[0].val
        );
        assertEq(
            attestationStation.attestations(alice_attestor, attestationData[1].about, attestationData[1].key),
            attestationData[1].val
        );
        assertEq(
            attestationStation.attestations(alice_attestor, attestationData[2].about, attestationData[2].key),
            attestationData[2].val
        );
    }
}
