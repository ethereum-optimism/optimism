// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { Semver } from "../universal/Semver.sol";
import { L2OutputOracle } from "./L2OutputOracle.sol";

contract OptimismSystemConfig is Ownable, Semver {

    L2OutputOracle immutable public l2OutputOracle;
    address public unsafeBlockSigner;

    event SetBatcher(address indexed batcher);
    event SetUnsafeBlockSigner(address indexed previous, address indexed next);

    constructor(
        L2OutputOracle _l2OutputOracle,
        address _unsafeBlockSigner
    ) Semver(0, 0, 1) {
        unsafeBlockSigner = _unsafeBlockSigner;
        l2OutputOracle = _l2OutputOracle;
    }

    function safeBlockSigner() external view returns (address) {
        return l2OutputOracle.proposer();
    }

    function setBatcher(address batcher) onlyOwner external {
        emit SetBatcher(batcher);
    }

    function setUnsafeBlockSigner(address _unsafeBlockSigner) onlyOwner external {
        address prev = unsafeBlockSigner;
        unsafeBlockSigner = _unsafeBlockSigner;
        emit SetUnsafeBlockSigner(prev, _unsafeBlockSigner);
    }
}
