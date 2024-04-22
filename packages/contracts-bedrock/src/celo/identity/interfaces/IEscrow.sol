// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

interface IEscrow {
    function transfer(
        bytes32 identifier,
        address token,
        uint256 value,
        uint256 expirySeconds,
        address paymentId,
        uint256 minAttestations
    )
        external
        returns (bool);
    function transferWithTrustedIssuers(
        bytes32 identifier,
        address token,
        uint256 value,
        uint256 expirySeconds,
        address paymentId,
        uint256 minAttestations,
        address[] calldata trustedIssuers
    )
        external
        returns (bool);
    function withdraw(address paymentID, uint8 v, bytes32 r, bytes32 s) external returns (bool);
    function revoke(address paymentID) external returns (bool);

    // view functions
    function getReceivedPaymentIds(bytes32 identifier) external view returns (address[] memory);
    function getSentPaymentIds(address sender) external view returns (address[] memory);
    function getTrustedIssuersPerPayment(address paymentId) external view returns (address[] memory);
    function getDefaultTrustedIssuers() external view returns (address[] memory);
    function MAX_TRUSTED_ISSUERS_PER_PAYMENT() external view returns (uint256);

    // onlyOwner functions
    function addDefaultTrustedIssuer(address trustedIssuer) external;
    function removeDefaultTrustedIssuer(address trustedIssuer, uint256 index) external;
}
