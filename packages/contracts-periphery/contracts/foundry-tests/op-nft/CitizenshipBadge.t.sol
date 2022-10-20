//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import "forge-std/Test.sol";
import { CitizenshipBadge } from "../../universal/op-nft/CitizenshipBadge.sol";
import { SocialContract } from "../../universal/op-nft/SocialContract.sol";
import { CitizenshipChecker } from "../../universal/op-nft/CitizenshipChecker.sol";
import "forge-std/console.sol";

contract CitizenshipBadgeTest is Test {
    using stdStorage for StdStorage;

    CitizenshipBadge private citizenshipBadge;
    SocialContract private socialContract;
    CitizenshipChecker private citizenshipChecker;

    function _setUp() public {}

    function test_mint() public {
        address admin = address(1);
        address opco = address(69);
        address bob = address(2);
        socialContract = new SocialContract();
        citizenshipChecker = new CitizenshipChecker(admin, address(socialContract));
        citizenshipBadge = new CitizenshipBadge(
            admin,
            address(socialContract),
            address(citizenshipChecker)
        );

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
        citizenshipBadge.mint(bob, proof);

        assertEq(citizenshipBadge.balanceOf(bob), 1);
    }
}
