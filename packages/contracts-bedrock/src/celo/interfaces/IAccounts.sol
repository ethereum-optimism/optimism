// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IAccounts {
    function isAccount(address) external view returns (bool);
    function voteSignerToAccount(address) external view returns (address);
    function validatorSignerToAccount(address) external view returns (address);
    function attestationSignerToAccount(address) external view returns (address);
    function signerToAccount(address) external view returns (address);
    function getAttestationSigner(address) external view returns (address);
    function getValidatorSigner(address) external view returns (address);
    function getVoteSigner(address) external view returns (address);
    function hasAuthorizedVoteSigner(address) external view returns (bool);
    function hasAuthorizedValidatorSigner(address) external view returns (bool);
    function hasAuthorizedAttestationSigner(address) external view returns (bool);

    function setAccountDataEncryptionKey(bytes calldata) external;
    function setMetadataURL(string calldata) external;
    function setName(string calldata) external;
    function setWalletAddress(address, uint8, bytes32, bytes32) external;
    function setAccount(string calldata, bytes calldata, address, uint8, bytes32, bytes32) external;

    function getDataEncryptionKey(address) external view returns (bytes memory);
    function getWalletAddress(address) external view returns (address);
    function getMetadataURL(address) external view returns (string memory);
    function batchGetMetadataURL(address[] calldata) external view returns (uint256[] memory, bytes memory);
    function getName(address) external view returns (string memory);

    function authorizeVoteSigner(address, uint8, bytes32, bytes32) external;
    function authorizeValidatorSigner(address, uint8, bytes32, bytes32) external;
    function authorizeValidatorSignerWithPublicKey(address, uint8, bytes32, bytes32, bytes calldata) external;
    function authorizeValidatorSignerWithKeys(
        address,
        uint8,
        bytes32,
        bytes32,
        bytes calldata,
        bytes calldata,
        bytes calldata
    )
        external;
    function authorizeAttestationSigner(address, uint8, bytes32, bytes32) external;
    function createAccount() external returns (bool);

    function setPaymentDelegation(address, uint256) external;
    function getPaymentDelegation(address) external view returns (address, uint256);
    function isSigner(address, address, bytes32) external view returns (bool);
}
