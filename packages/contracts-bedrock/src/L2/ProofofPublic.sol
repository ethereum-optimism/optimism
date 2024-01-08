// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;


contract ProofofPublic{

    mapping(address => bytes)  proofs;
    function SubmitProof(bytes calldata proofData) public {
        proofs[msg.sender]=proofData;
    }

    function getRandomUser() public view returns (address) {
        return address(this);
    }
}