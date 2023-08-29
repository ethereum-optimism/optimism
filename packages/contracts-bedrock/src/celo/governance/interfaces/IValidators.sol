// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface IValidators {
    function registerValidator(bytes calldata, bytes calldata, bytes calldata) external returns (bool);
    function deregisterValidator(uint256) external returns (bool);
    function affiliate(address) external returns (bool);
    function deaffiliate() external returns (bool);
    function updateBlsPublicKey(bytes calldata, bytes calldata) external returns (bool);
    function registerValidatorGroup(uint256) external returns (bool);
    function deregisterValidatorGroup(uint256) external returns (bool);
    function addMember(address) external returns (bool);
    function addFirstMember(address, address, address) external returns (bool);
    function removeMember(address) external returns (bool);
    function reorderMember(address, address, address) external returns (bool);
    function updateCommission() external;
    function setNextCommissionUpdate(uint256) external;
    function resetSlashingMultiplier() external;

    // only owner
    function setCommissionUpdateDelay(uint256) external;
    function setMaxGroupSize(uint256) external returns (bool);
    function setMembershipHistoryLength(uint256) external returns (bool);
    function setValidatorScoreParameters(uint256, uint256) external returns (bool);
    function setGroupLockedGoldRequirements(uint256, uint256) external returns (bool);
    function setValidatorLockedGoldRequirements(uint256, uint256) external returns (bool);
    function setSlashingMultiplierResetPeriod(uint256) external;

    // view functions
    function getMaxGroupSize() external view returns (uint256);
    function getCommissionUpdateDelay() external view returns (uint256);
    function getValidatorScoreParameters() external view returns (uint256, uint256);
    function getMembershipHistory(address)
        external
        view
        returns (uint256[] memory, address[] memory, uint256, uint256);
    function calculateEpochScore(uint256) external view returns (uint256);
    function calculateGroupEpochScore(uint256[] calldata) external view returns (uint256);
    function getAccountLockedGoldRequirement(address) external view returns (uint256);
    function meetsAccountLockedGoldRequirements(address) external view returns (bool);
    function getValidatorBlsPublicKeyFromSigner(address) external view returns (bytes memory);
    function getValidator(address account)
        external
        view
        returns (bytes memory, bytes memory, address, uint256, address);
    function getValidatorGroup(address)
        external
        view
        returns (address[] memory, uint256, uint256, uint256, uint256[] memory, uint256, uint256);
    function getGroupNumMembers(address) external view returns (uint256);
    function getTopGroupValidators(address, uint256) external view returns (address[] memory);
    function getGroupsNumMembers(address[] calldata accounts) external view returns (uint256[] memory);
    function getNumRegisteredValidators() external view returns (uint256);
    function groupMembershipInEpoch(address, uint256, uint256) external view returns (address);

    // only registered contract
    function updateEcdsaPublicKey(address, address, bytes calldata) external returns (bool);
    function updatePublicKeys(
        address,
        address,
        bytes calldata,
        bytes calldata,
        bytes calldata
    )
        external
        returns (bool);
    function getValidatorLockedGoldRequirements() external view returns (uint256, uint256);
    function getGroupLockedGoldRequirements() external view returns (uint256, uint256);
    function getRegisteredValidators() external view returns (address[] memory);
    function getRegisteredValidatorSigners() external view returns (address[] memory);
    function getRegisteredValidatorGroups() external view returns (address[] memory);
    function isValidatorGroup(address) external view returns (bool);
    function isValidator(address) external view returns (bool);
    function getValidatorGroupSlashingMultiplier(address) external view returns (uint256);
    function getMembershipInLastEpoch(address) external view returns (address);
    function getMembershipInLastEpochFromSigner(address) external view returns (address);

    // only VM
    function updateValidatorScoreFromSigner(address, uint256) external;
    function distributeEpochPaymentsFromSigner(address, uint256) external returns (uint256);

    // only slasher
    function forceDeaffiliateIfValidator(address) external;
    function halveSlashingMultiplier(address) external;
}
