// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "test/setup/Test.sol";
import { AttestedEventVerifier } from "src/private-interop/AttestedEventVerifier.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @notice Exercises signature binding and consumption for the minimal trusted-signer verifier.
contract AttestedEventVerifier_VerifyAndConsumeEvent_Test is Test {
    AttestedEventVerifier internal verifier;
    Identifier internal id;
    address internal inbox = address(0x1234);
    uint256 internal signerKey = 123;
    bytes32 internal payloadHash = keccak256("event");

    function setUp() public {
        verifier = new AttestedEventVerifier(vm.addr(signerKey), inbox);
        id = Identifier(address(0x5678), 1, 2, 3, block.chainid);
    }

    function _proof() internal view returns (bytes memory) {
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(signerKey, verifier.eventDigest(id, payloadHash));
        return abi.encodePacked(r, s, v);
    }

    function test_verifyAndConsumeEvent_validProof_succeeds() external {
        bytes memory proof = _proof();
        vm.prank(inbox);
        assertTrue(verifier.verifyAndConsumeEvent(id, payloadHash, proof));
        vm.prank(inbox);
        assertFalse(verifier.verifyAndConsumeEvent(id, payloadHash, proof));
    }

    function test_verifyAndConsumeEvent_changedPayload_fails() external {
        bytes memory proof = _proof();
        vm.prank(inbox);
        assertFalse(verifier.verifyAndConsumeEvent(id, bytes32(uint256(1)), proof));
        assertFalse(verifier.consumedEvents(keccak256(abi.encode(id))));
    }

    function test_verifyAndConsumeEvent_changedPosition_fails() external {
        bytes memory proof = _proof();
        id.logIndex++;
        vm.prank(inbox);
        assertFalse(verifier.verifyAndConsumeEvent(id, payloadHash, proof));
    }

    function test_verifyAndConsumeEvent_wrongCaller_fails() external {
        assertFalse(verifier.verifyAndConsumeEvent(id, payloadHash, _proof()));
    }

    function test_verifyAndConsumeEvent_otherVerifier_fails() external {
        AttestedEventVerifier other = new AttestedEventVerifier(vm.addr(signerKey), inbox);
        bytes memory proof = _proof();
        vm.prank(inbox);
        assertFalse(other.verifyAndConsumeEvent(id, payloadHash, proof));
    }

    function test_verifyAndConsumeEvent_otherChain_fails() external {
        bytes memory proof = _proof();
        vm.chainId(block.chainid + 1);
        vm.prank(inbox);
        assertFalse(verifier.verifyAndConsumeEvent(id, payloadHash, proof));
    }

    function test_verifyAndConsumeEvent_malformedProof_fails() external {
        vm.prank(inbox);
        assertFalse(verifier.verifyAndConsumeEvent(id, payloadHash, hex"1234"));
    }
}
