// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { SystemConfig } from "./SystemConfig.sol";

/**
 * @title BaseSystemDictator
 * @notice The BaseSystemDictator is a base contract for SystemDictator contracts.
 */
contract BaseSystemDictator is Ownable {
    /**
     * @notice System configuration.
     */
    SystemConfig public config;

    /**
     * @notice Current step;
     */
    uint256 public currentStep = 1;

    /**
     * @notice Checks that the current step is the expected step, then bumps the current step.
     *
     * @param _step Current step.
     */
    modifier step(uint256 _step) {
        require(currentStep == _step, "BaseSystemDictator: incorrect step");
        _;
        currentStep++;
    }

    /**
     * @param _config System configuration.
     */
    constructor(SystemConfig memory _config) Ownable() {
        config = _config;
        _transferOwnership(config.globalConfig.controller);
    }
}
