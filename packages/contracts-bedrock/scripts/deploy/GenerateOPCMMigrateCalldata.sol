// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOPContractsManagerInteropMigrator, IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { Claim, Duration, Proposal, Hash } from "src/dispute/lib/Types.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

/// @title GenerateOPCMMigrateCalldata
/// @notice Script to generate the calldata for the OPCM.migrate function. Useful for constructing public devnets.
/// @dev Usage: forge script ./scripts/deploy/GenerateOPCMMigrateCalldata.sol --sig 'run(string)'
/// ./deploy-config/opcm-migrate-config.json
/// Due to foundry file access restrictions, the opcm-migrate-config.json file must be located in the deploy-config
/// directory located at foundry root.
/// Config example:
///  {
///      "absolutePrestate": "0x1234567890abcdef1234567890abcdef12345678",
///      "usePermissionlessGame": true,
///      "startingAnchorRoot": {
///          "root": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
///          "l2SequenceNumber": 123456789
///      },
///      "proposer": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
///      "challenger": "0x1234567890abcdef1234567890abcdef12345678",
///      "maxGameDepth": 73,
///      "splitDepth": 30,
///      "initBond": 80000000000000000,
///      "clockExtension": 10800,
///      "maxClockDuration": 302400,
///      "opChainConfigs": [
///          {
///              "systemConfigProxy": "0x1234567890abcdef1234567890abcdef12345678",
///              "proxyAdmin": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
///          },
///          {
///              "systemConfigProxy": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
///              "proxyAdmin": "0x1234567890abcdef1234567890abcdef12345678"
///          }
///      ]
///  }
contract GenerateOPCMMigrateCalldata is Script {
    bytes32 absolutePrestate;
    bool usePermissionlessGame;
    Proposal startingAnchorRoot;
    address proposer;
    address challenger;
    uint64 maxGameDepth;
    uint64 splitDepth;
    uint256 initBond;
    Duration clockExtension;
    Duration maxClockDuration;

    function readConfig(string memory _configFile)
        internal
        returns (IOPContractsManagerInteropMigrator.MigrateInput memory)
    {
        string memory json;
        try vm.readFile(_configFile) returns (string memory json_) {
            json = json_;
        } catch {
            require(false, "GenerateOPCMMigrateCalldata: Failed to read config file");
        }

        absolutePrestate = stdJson.readBytes32(json, "$.absolutePrestate");
        usePermissionlessGame = stdJson.readBool(json, "$.usePermissionlessGame");
        startingAnchorRoot = Proposal({
            root: Hash.wrap(stdJson.readBytes32(json, "$.startingAnchorRoot.root")),
            l2SequenceNumber: stdJson.readUint(json, "$.startingAnchorRoot.l2SequenceNumber")
        });
        proposer = stdJson.readAddress(json, "$.proposer");
        challenger = stdJson.readAddress(json, "$.challenger");
        maxGameDepth = uint64(stdJson.readUint(json, "$.maxGameDepth"));
        splitDepth = uint64(stdJson.readUint(json, "$.splitDepth"));
        initBond = stdJson.readUint(json, "$.initBond");
        clockExtension = Duration.wrap(uint64(stdJson.readUint(json, "$.clockExtension")));
        maxClockDuration = Duration.wrap(uint64(stdJson.readUint(json, "$.maxClockDuration")));

        IOPContractsManager.OpChainConfig[] memory opChainConfigs = new IOPContractsManager.OpChainConfig[](2);
        opChainConfigs[0] = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(stdJson.readAddress(json, "$.opChainConfigs[0].systemConfigProxy")),
            proxyAdmin: IProxyAdmin(stdJson.readAddress(json, "$.opChainConfigs[0].proxyAdmin")),
            absolutePrestate: Claim.wrap(absolutePrestate)
        });
        opChainConfigs[1] = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(stdJson.readAddress(json, "$.opChainConfigs[1].systemConfigProxy")),
            proxyAdmin: IProxyAdmin(stdJson.readAddress(json, "$.opChainConfigs[1].proxyAdmin")),
            absolutePrestate: Claim.wrap(absolutePrestate)
        });

        return IOPContractsManagerInteropMigrator.MigrateInput({
            usePermissionlessGame: usePermissionlessGame,
            startingAnchorRoot: startingAnchorRoot,
            gameParameters: IOPContractsManagerInteropMigrator.GameParameters({
                proposer: proposer,
                challenger: challenger,
                maxGameDepth: maxGameDepth,
                splitDepth: splitDepth,
                initBond: initBond,
                clockExtension: clockExtension,
                maxClockDuration: maxClockDuration
            }),
            opChainConfigs: opChainConfigs
        });
    }

    function run(string memory _configFile) public {
        IOPContractsManagerInteropMigrator.MigrateInput memory inputs = readConfig(_configFile);
        bytes memory cd = abi.encodeCall(IOPContractsManager.migrate, (inputs));
        console.log("OPCM.migrate calldata: ");
        console.logBytes(cd);
    }
}
