// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console } from "forge-std/console.sol";
import { CommonBase } from "forge-std/Base.sol";

abstract contract Simulator is CommonBase {
    struct SimulationStateOverride {
        address contractAddress;
        SimulationStorageOverride[] overrides;
    }

    struct SimulationStorageOverride {
        bytes32 key;
        bytes32 value;
    }

    function overrideSafeThreshold(address _safe) public pure returns (SimulationStateOverride memory) {
        SimulationStorageOverride[] memory overrides = new SimulationStorageOverride[](1);
        // set the threshold (slot 4) to 1
        overrides[0] = SimulationStorageOverride({
            key: bytes32(uint256(0x4)),
            value: bytes32(uint256(0x1))
        });
        return SimulationStateOverride({
            contractAddress: _safe,
            overrides: overrides
        });
    }

    function overrideSafeThresholdAndOwner(address _safe, address _owner) public pure returns (SimulationStateOverride memory) {
        SimulationStorageOverride[] memory overrides = new SimulationStorageOverride[](4);

        // set the threshold (slot 4) to 1
        overrides[0] = SimulationStorageOverride({
            key: bytes32(uint256(0x4)),
            value: bytes32(uint256(0x1))
        });

        // set the ownerCount (slot 3) to 1
        overrides[1] = SimulationStorageOverride({
            key: bytes32(uint256(0x3)),
            value: bytes32(uint256(0x1))
        });

        // override the owner mapping (slot 2), which requires two key/value pairs: { 0x1: _owner, _owner: 0x1 }
        overrides[2] = SimulationStorageOverride({
            key: bytes32(0xe90b7bceb6e7df5418fb78d8ee546e97c83a08bbccc01a0644d599ccd2a7c2e0), // keccak256(1 || 2)
            value: bytes32(uint256(uint160(_owner)))
        });
        overrides[3] = SimulationStorageOverride({
            key: keccak256(abi.encode(_owner, uint256(2))),
            value: bytes32(uint256(0x1))
        });

        return SimulationStateOverride({
            contractAddress: _safe,
            overrides: overrides
        });
    }

    function logSimulationLink(address _to, bytes memory _data, address _from) public view {
        logSimulationLink(_to, _data, _from, new SimulationStateOverride[](0));
    }

    function logSimulationLink(address _to, bytes memory _data, address _from, SimulationStateOverride[] memory _overrides) public view {
        (, bytes memory projData) = VM_ADDRESS.staticcall(
            abi.encodeWithSignature("envOr(string,string)", "TENDERLY_PROJECT", "TENDERLY_PROJECT")
        );
        string memory proj = abi.decode(projData, (string));

        (, bytes memory userData) = VM_ADDRESS.staticcall(
            abi.encodeWithSignature("envOr(string,string)", "TENDERLY_USERNAME", "TENDERLY_USERNAME")
        );
        string memory username = abi.decode(userData, (string));

        // the following characters are url encoded: []{}
        string memory stateOverrides = "%5B";
        for (uint256 i; i < _overrides.length; i++) {
            SimulationStateOverride memory _override = _overrides[i];
            if (i > 0) stateOverrides = string.concat(stateOverrides, ",");
            stateOverrides = string.concat(
                stateOverrides,
                "%7B\"contractAddress\":\"",
                vm.toString(_override.contractAddress),
                "\",\"storage\":%5B"
            );
            for (uint256 j; j < _override.overrides.length; j++) {
                if (j > 0) stateOverrides = string.concat(stateOverrides, ",");
                stateOverrides = string.concat(
                    stateOverrides,
                    "%7B\"key\":\"",
                    vm.toString(_override.overrides[j].key),
                    "\",\"value\":\"",
                    vm.toString(_override.overrides[j].value),
                    "\"%7D"
                );
            }
            stateOverrides = string.concat(stateOverrides, "%5D%7D");
        }
        stateOverrides = string.concat(stateOverrides, "%5D");

        string memory str = string.concat(
            "https://dashboard.tenderly.co/",
            username,
            "/",
            proj,
            "/simulator/new?network=",
            vm.toString(block.chainid),
            "&contractAddress=",
            vm.toString(_to),
            "&from=",
            vm.toString(_from),
            "&stateOverrides=",
            stateOverrides
        );
        if (bytes(str).length + _data.length * 2 > 7980) {
            // tenderly's nginx has issues with long URLs, so print the raw input data separately
            str = string.concat(str, "\nInsert the following hex into the 'Raw input data' field:");
            console.log(str);
            console.log(vm.toString(_data));
        } else {
            str = string.concat(str, "&rawFunctionInput=", vm.toString(_data));
            console.log(str);
        }
    }
}
