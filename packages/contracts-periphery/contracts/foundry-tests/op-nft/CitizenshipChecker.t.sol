// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "forge-std/Test.sol";
import { CitizenshipChecker } from "../../universal/op-nft/CitizenshipChecker.sol";
import { SocialContract } from "../../universal/op-nft/SocialContract.sol";

contract CitizenCheckerTest is Test {
    CitizenshipChecker private citizenshipChecker;
    SocialContract private socialContract;

    event AttestationCreated(
        address indexed creator,
        address indexed about,
        bytes32 indexed key,
        bytes val
    );

    function setUp() public {
        socialContract = new SocialContract();
        citizenshipChecker = new CitizenshipChecker(address(1), address(socialContract));
        vm.label(address(citizenshipChecker), "CitizenChecker");
        vm.label(address(socialContract), "SocialContract");
    }

    function test_isCitizen() external {
        //
        address admin = address(1);
        address opco = address(69);
        address bob = address(2);

        // Have admin attest that opco is opco
        SocialContract.AttestationData[]
            memory attestationData = new SocialContract.AttestationData[](1);
        attestationData[0] = SocialContract.AttestationData({
            about: opco,
            key: keccak256("op.opco"),
            val: abi.encodePacked(uint256(100)) //abi.encodePacked(true)
        });
        vm.prank(admin);
        socialContract.attest(attestationData);

        assertEq(
            socialContract.attestations(admin, opco, keccak256("op.opco")),
            abi.encodePacked(uint256(100))
        );

        // Have opco attest bob is a citizen
        SocialContract.AttestationData[]
            memory attestationData2 = new SocialContract.AttestationData[](1);
        attestationData2[0] = SocialContract.AttestationData({
            about: bob,
            key: keccak256(abi.encodePacked("op.opco.citizen", uint256(1))),
            val: abi.encode(true)
        });
        vm.prank(opco);
        socialContract.attest(attestationData2);

        assertEq(
            socialContract.attestations(
                opco,
                bob,
                keccak256(abi.encodePacked("op.opco.citizen", uint256(1)))
            ),
            abi.encode(true)
        );
        bytes memory proof = abi.encode(opco, uint256(1));
        assertTrue(citizenshipChecker.isCitizen(bob, proof));
    }

    function testFail_isCitizen2() external {
        address opco = address(69);
        address charlie = address(4);

        bytes memory proof = abi.encode(opco, uint256(1));
        assertFalse(citizenshipChecker.isCitizen(charlie, proof));
        citizenshipChecker.isCitizen(charlie, proof);
    }

    function test_attestEmitsEvent() public {
        address opco = address(69);
        address alice = address(2);
        SocialContract.AttestationData[] memory attestations = new SocialContract.AttestationData[](
            2
        );
        attestations[0] = SocialContract.AttestationData({
            about: address(this),
            key: keccak256("op.opco"),
            val: abi.encode(true)
        });
        attestations[1] = SocialContract.AttestationData({
            about: alice,
            key: keccak256("op.opco.citizen"),
            val: abi.encode(true)
        });
        vm.expectEmit(true, true, true, true);
        emit AttestationCreated(
            address(this),
            alice,
            keccak256("op.opco.citizen"),
            abi.encode(true)
        );
        socialContract.attest(attestations);
    }
}
