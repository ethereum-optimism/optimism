// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

interface ICeloVersionedContract {
    /**
     * @notice Returns the storage, major, minor, and patch version of the contract.
     * @return Storage version of the contract.
     * @return Major version of the contract.
     * @return Minor version of the contract.
     * @return Patch version of the contract.
     */
    function getVersionNumber() external pure returns (uint256, uint256, uint256, uint256);
}
