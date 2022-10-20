// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { SocialContract } from "../../universal/op-nft/SocialContract.sol";

contract SocialContractTest is Test {
    SocialContract sc;

    function setUp() public {
        sc = new SocialContract();
        vm.label(address(sc), "SocialContract");
    }

    function test_attest() external {
        bytes memory proof = hex"";
        SocialContract.AttestationData[] memory attestations = new SocialContract.AttestationData[](
            1
        );
        attestations[0] = SocialContract.AttestationData({
            about: address(this),
            key: keccak256("key"),
            val: proof
        });

        sc.attest(attestations);
        assertEq(sc.attestations(address(this), address(this), keccak256("key")), proof);
    }

    function test_attestationEvent() external {
        bytes memory proof = hex"";
        SocialContract.AttestationData[] memory attestations = new SocialContract.AttestationData[](
            1
        );
        attestations[0] = SocialContract.AttestationData({
            about: address(this),
            key: keccak256("key"),
            val: proof
        });

        sc.attest(attestations);
    }
}
