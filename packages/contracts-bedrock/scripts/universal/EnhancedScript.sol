// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { console } from "forge-std/console.sol";
import { Script } from "forge-std/Script.sol";
import { Semver } from "../../contracts/universal/Semver.sol";

/// @title EnhancedScript
/// @notice Enhances forge-std' Script.sol with some additional application-specific functionality.
///         Logs simulation links using Tenderly.
abstract contract EnhancedScript is Script {
    /// @notice Helper function used to compute the hash of Semver's version string to be used in a
    ///         comparison.
    function _versionHash(address _addr) internal view returns (bytes32) {
        return keccak256(bytes(Semver(_addr).version()));
    }

    /// @notice Log a tenderly simulation link. The TENDERLY_USERNAME and TENDERLY_PROJECT
    ///         environment variables will be used if they are present. The vm is staticcall'ed
    ///         because of a compiler issue with the higher level ABI.
    function logSimulationLink(address _to, bytes memory _data, address _from) public view {
        (, bytes memory projData) = VM_ADDRESS.staticcall(
            abi.encodeWithSignature("envOr(string,string)", "TENDERLY_PROJECT", "TENDERLY_PROJECT")
        );
        string memory proj = abi.decode(projData, (string));

        (, bytes memory userData) = VM_ADDRESS.staticcall(
            abi.encodeWithSignature("envOr(string,string)", "TENDERLY_USERNAME", "TENDERLY_USERNAME")
        );
        string memory username = abi.decode(userData, (string));

        string memory str = string.concat(
            "https://dashboard.tenderly.co/",
            username,
            "/",
            proj,
            "/simulator/new?network=",
            vm.toString(block.chainid),
            "&contractAddress=",
            vm.toString(_to),
            "&rawFunctionInput=",
            vm.toString(_data),
            "&from=",
            vm.toString(_from)
        );
        console.log(str);
    }
}
