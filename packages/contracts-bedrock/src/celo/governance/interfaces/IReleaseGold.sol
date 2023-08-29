// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IReleaseGold {
    function transfer(address, uint256) external;
    function unlockGold(uint256) external;
    function withdrawLockedGold(uint256) external;
    function authorizeVoteSigner(address payable, uint8, bytes32, bytes32) external;
    function authorizeValidatorSigner(address payable, uint8, bytes32, bytes32) external;
    function authorizeValidatorSignerWithPublicKey(address payable, uint8, bytes32, bytes32, bytes calldata) external;
    function authorizeValidatorSignerWithKeys(
        address payable,
        uint8,
        bytes32,
        bytes32,
        bytes calldata,
        bytes calldata,
        bytes calldata
    )
        external;
    function authorizeAttestationSigner(address payable, uint8, bytes32, bytes32) external;
    function revokeActive(address, uint256, address, address, uint256) external;
    function revokePending(address, uint256, address, address, uint256) external;

    // view functions
    function getTotalBalance() external view returns (uint256);
    function getRemainingTotalBalance() external view returns (uint256);
    function getRemainingUnlockedBalance() external view returns (uint256);
    function getRemainingLockedBalance() external view returns (uint256);
    function getCurrentReleasedTotalAmount() external view returns (uint256);
    function isRevoked() external view returns (bool);

    // only beneficiary
    function setCanExpire(bool) external;
    function withdraw(uint256) external;
    function lockGold(uint256) external;
    function relockGold(uint256, uint256) external;
    function setAccount(string calldata, bytes calldata, address, uint8, bytes32, bytes32) external;
    function createAccount() external;
    function setAccountName(string calldata) external;
    function setAccountWalletAddress(address, uint8, bytes32, bytes32) external;
    function setAccountDataEncryptionKey(bytes calldata) external;
    function setAccountMetadataURL(string calldata) external;

    // only owner
    function setBeneficiary(address payable) external;

    // only release owner
    function setLiquidityProvision() external;
    function setMaxDistribution(uint256) external;
    function refundAndFinalize() external;
    function revoke() external;
    function expire() external;
}
