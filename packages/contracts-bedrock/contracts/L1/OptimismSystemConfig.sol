// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { Semver } from "../universal/Semver.sol";

contract OptimismSystemConfig is Ownable, Semver {

    address public unsafeBlockSigner;

    event SetBatcher(address indexed batcher);
    event SetUnsafeBlockSigner(address indexed previous, address indexed next);

    constructor(address _unsafeBlockSigner) Semver(0, 0, 1) {
        unsafeBlockSigner = _unsafeBlockSigner;
    }

    function setBatcher(address batcher) onlyOwner public {
        emit SetBatcher(batcher);
    }

    function setUnsafeBlockSigner(address _unsafeBlockSigner) public {
        address prev = unsafeBlockSigner;
        unsafeBlockSigner = _unsafeBlockSigner;
        emit SetUnsafeBlockSigner(prev, _unsafeBlockSigner);
    }
}
