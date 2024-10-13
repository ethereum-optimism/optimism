// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

contract L1OptimismMintableERC20Factory is OptimismMintableERC20Factory, Initializable {
    address internal standardBridge;

    constructor() {
        _disableInitializers();
    }

    function initialize(address _bridge) public initializer {
        standardBridge = _bridge;
    }

    function bridge() public view virtual override returns (address) {
        return standardBridge;
    }
}
