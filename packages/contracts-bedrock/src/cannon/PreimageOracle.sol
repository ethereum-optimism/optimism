// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IPreimageOracle } from "./interfaces/IPreimageOracle.sol";
import { PreimageKeyLib } from "./PreimageKeyLib.sol";
import { LibKeccak } from "@lib-keccak/LibKeccak.sol";
import "src/cannon/libraries/CannonErrors.sol";
import "src/cannon/libraries/CannonTypes.sol";

/// @title PreimageOracle
/// @notice A contract for storing permissioned pre-images.
/// @custom:attribution Solady <https://github.com/Vectorized/solady/blob/main/src/utils/MerkleProofLib.sol#L13-L43>
/// @custom:attribution Beacon Deposit Contract <0x00000000219ab540356cbb839cbe05303d7705fa>
contract PreimageOracle { }
