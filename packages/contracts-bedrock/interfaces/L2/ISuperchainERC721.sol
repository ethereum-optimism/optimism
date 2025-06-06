// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ICrosschainERC721 } from "interfaces/L2/ICrosschainERC721.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title ISuperchainERC721
/// @notice This interface is available on the SuperchainERC721 contract.
/// @dev This interface is needed for the abstract SuperchainERC721 implementation but is not part of the standard
interface ISuperchainERC721 is ICrosschainERC721, IERC721, ISemver {
    error Unauthorized();

    function supportsInterface(bytes4 _interfaceId) external view returns (bool);

    function __constructor__() external;
}
