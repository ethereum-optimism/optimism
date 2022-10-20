// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

contract SystemConfig is Ownable {
    // Version 0 schema
    uint256 public constant VERSION = 0;

    uint256 public overhead;
    uint256 public scalar;
    address public batcher;

    event ConfigUpdate(
        uint256 indexed version,
        UpdateType indexed updateType,
        bytes data
    );

    enum UpdateType {
        BATCHER,
        GAS_CONFIG
    }

    constructor(address _owner) {
        transferOwnership(_owner);
    }

    function setBatcher(address _batcher) external onlyOwner {
        batcher = _batcher;

        bytes memory data = abi.encode(_batcher);
        emit ConfigUpdate(VERSION, UpdateType.BATCHER, data);
    }

    function setGasConfig(uint256 _overhead, uint256 _scalar) external onlyOwner {
        overhead = _overhead;
        scalar = _scalar;

        bytes memory data = abi.encode(_overhead, _scalar);
        emit ConfigUpdate(VERSION, UpdateType.GAS_CONFIG, data);
    }
}
