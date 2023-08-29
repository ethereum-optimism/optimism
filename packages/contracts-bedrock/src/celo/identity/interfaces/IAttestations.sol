// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IAttestations {
    function revoke(bytes32, uint256) external;
    function withdraw(address) external;

    // view functions
    function getUnselectedRequest(bytes32, address) external view returns (uint32, uint32, address);
    function getAttestationIssuers(bytes32, address) external view returns (address[] memory);
    function getAttestationStats(bytes32, address) external view returns (uint32, uint32);
    function batchGetAttestationStats(bytes32[] calldata)
        external
        view
        returns (uint256[] memory, address[] memory, uint64[] memory, uint64[] memory);
    function getAttestationState(bytes32, address, address) external view returns (uint8, uint32, address);
    function getCompletableAttestations(
        bytes32,
        address
    )
        external
        view
        returns (uint32[] memory, address[] memory, uint256[] memory, bytes memory);
    function getAttestationRequestFee(address) external view returns (uint256);
    function getMaxAttestations() external view returns (uint256);
    function validateAttestationCode(bytes32, address, uint8, bytes32, bytes32) external view returns (address);
    function lookupAccountsForIdentifier(bytes32) external view returns (address[] memory);
    function requireNAttestationsRequested(bytes32, address, uint32) external view;

    // only owner
    function setAttestationRequestFee(address, uint256) external;
    function setAttestationExpiryBlocks(uint256) external;
    function setSelectIssuersWaitBlocks(uint256) external;
    function setMaxAttestations(uint256) external;
}
