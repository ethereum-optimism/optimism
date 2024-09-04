// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IOwnable } from "src/universal/interfaces/IOwnable.sol";
import { IGovernanceToken } from "./IGovernanceToken.sol";

// @title IMintManager
// @notice Interface for the MintManager contract.
interface IMintManager is IOwnable {
    function DENOMINATOR() external view returns (uint256);
    function MINT_CAP() external view returns (uint256);
    function MINT_PERIOD() external view returns (uint256);
    function governanceToken() external view returns (IGovernanceToken);
    function mint(address _account, uint256 _amount) external;
    function mintPermittedAfter() external view returns (uint256);
    function upgrade(address _newMintManager) external;
}
