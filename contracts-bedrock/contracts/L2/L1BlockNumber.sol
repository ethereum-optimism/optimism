//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { L1Block } from "./L1Block.sol";
import { Lib_BedrockPredeployAddresses } from "../libraries/Lib_BedrockPredeployAddresses.sol";

/**
 * @title L1BlockNumber
 * @dev L1BlockNumber is a legacy contract that fills the roll of the OVM_L1BlockNumber contract in
 * the old version of the Optimism system. Only necessary for backwards compatibility. If you want
 * to access the L1 block number going forward, you should use the L1Block contract instead.
 *
 * ADDRESS: 0x4200000000000000000000000000000000000013
 */
contract L1BlockNumber {
    receive() external payable {
        uint256 l1BlockNumber = getL1BlockNumber();
        assembly {
            mstore(0, l1BlockNumber)
            return(0, 32)
        }
    }

    fallback() external payable {
        uint256 l1BlockNumber = getL1BlockNumber();
        assembly {
            mstore(0, l1BlockNumber)
            return(0, 32)
        }
    }

    function getL1BlockNumber() public view returns (uint256) {
        return L1Block(Lib_BedrockPredeployAddresses.L1_BLOCK_ATTRIBUTES).number();
    }
}
