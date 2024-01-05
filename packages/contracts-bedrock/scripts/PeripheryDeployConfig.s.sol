// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

/// @title PeripheryDeployConfig
/// @notice Represents the configuration required to deploy the periphery contracts. It is expected
///         to read the file from JSON. A future improvement would be to have fallback
///         values if they are not defined in the JSON themselves.
contract PeripheryDeployConfig is Script {
    string internal _json;

    address public faucetAdmin;
    address public faucetDrippieOwner;
    uint256 public faucetDripV1Value;
    uint256 public faucetDripV1Interval;
    uint256 public faucetDripV1Threshold;
    uint256 public faucetDripV2Value;
    uint256 public faucetDripV2Interval;
    uint256 public faucetDripV2Threshold;
    uint256 public faucetAdminDripV1Value;
    uint256 public faucetAdminDripV1Interval;
    uint256 public faucetAdminDripV1Threshold;
    address public faucetGelatoTreasury;
    address public faucetGelatoRecipient;
    uint256 public faucetGelatoBalanceV1DripInterval;
    uint256 public faucetGelatoBalanceV1Value;
    uint256 public faucetGelatoThreshold;
    address public faucetOnchainAuthModuleAdmin;
    uint256 public faucetOnchainAuthModuleTtl;
    uint256 public faucetOnchainAuthModuleAmount;
    address public faucetOffchainAuthModuleAdmin;
    uint256 public faucetOffchainAuthModuleTtl;
    uint256 public faucetOffchainAuthModuleAmount;
    bool public installOpChainFaucetsDrips;
    bool public archivePreviousOpChainFaucetsDrips;
    uint256 public smallOpChainFaucetDripValue;
    uint256 public smallOpChainFaucetDripInterval;
    uint256 public largeOpChainFaucetDripValue;
    uint256 public largeOpChainFaucetDripInterval;
    uint256 public opChainAdminWalletDripValue;
    uint256 public opChainAdminWalletDripInterval;
    address public opL1BridgeAddress;
    address public baseL1BridgeAddress;
    address public zoraL1BridgeAddress;
    address public pgnL1BridgeAddress;
    address public orderlyL1BridgeAddress;
    address public modeL1BridgeAddress;
    address public lyraL1BridgeAddress;
    address[5] public smallFaucetsL1BridgeAddresses;
    address[2] public largeFaucetsL1BridgeAddresses;
    uint256 public dripVersion;
    uint256 public previousDripVersion;

    constructor(string memory _path) {
        console.log("PeripheryDeployConfig: reading file %s", _path);
        try vm.readFile(_path) returns (string memory data) {
            _json = data;
        } catch {
            console.log("Warning: unable to read config. Do not deploy unless you are not using config.");
            return;
        }

        faucetAdmin = stdJson.readAddress(_json, "$.faucetAdmin");
        faucetDrippieOwner = stdJson.readAddress(_json, "$.faucetDrippieOwner");
        faucetDripV1Value = stdJson.readUint(_json, "$.faucetDripV1Value");
        faucetDripV1Interval = stdJson.readUint(_json, "$.faucetDripV1Interval");
        faucetDripV1Threshold = stdJson.readUint(_json, "$.faucetDripV1Threshold");
        faucetDripV2Value = stdJson.readUint(_json, "$.faucetDripV2Value");
        faucetDripV2Interval = stdJson.readUint(_json, "$.faucetDripV2Interval");
        faucetDripV2Threshold = stdJson.readUint(_json, "$.faucetDripV2Threshold");
        faucetAdminDripV1Value = stdJson.readUint(_json, "$.faucetAdminDripV1Value");
        faucetAdminDripV1Interval = stdJson.readUint(_json, "$.faucetAdminDripV1Interval");
        faucetAdminDripV1Threshold = stdJson.readUint(_json, "$.faucetAdminDripV1Threshold");
        faucetGelatoTreasury = stdJson.readAddress(_json, "$.faucetGelatoTreasury");
        faucetGelatoRecipient = stdJson.readAddress(_json, "$.faucetGelatoRecipient");
        faucetGelatoBalanceV1DripInterval = stdJson.readUint(_json, "$.faucetGelatoBalanceV1DripInterval");
        faucetGelatoBalanceV1Value = stdJson.readUint(_json, "$.faucetGelatoBalanceV1Value");
        faucetGelatoThreshold = stdJson.readUint(_json, "$.faucetGelatoThreshold");
        faucetOnchainAuthModuleAdmin = stdJson.readAddress(_json, "$.faucetOnchainAuthModuleAdmin");
        faucetOnchainAuthModuleTtl = stdJson.readUint(_json, "$.faucetOnchainAuthModuleTtl");
        faucetOnchainAuthModuleAmount = stdJson.readUint(_json, "$.faucetOnchainAuthModuleAmount");
        faucetOffchainAuthModuleAdmin = stdJson.readAddress(_json, "$.faucetOffchainAuthModuleAdmin");
        faucetOffchainAuthModuleTtl = stdJson.readUint(_json, "$.faucetOffchainAuthModuleTtl");
        faucetOffchainAuthModuleAmount = stdJson.readUint(_json, "$.faucetOffchainAuthModuleAmount");
        installOpChainFaucetsDrips = stdJson.readBool(_json, "$.installOpChainFaucetsDrips");
        archivePreviousOpChainFaucetsDrips = stdJson.readBool(_json, "$.archivePreviousOpChainFaucetsDrips");
        opL1BridgeAddress = stdJson.readAddress(_json, "$.opL1BridgeAddress");
        baseL1BridgeAddress = stdJson.readAddress(_json, "$.baseL1BridgeAddress");
        zoraL1BridgeAddress = stdJson.readAddress(_json, "$.zoraL1BridgeAddress");
        pgnL1BridgeAddress = stdJson.readAddress(_json, "$.pgnL1BridgeAddress");
        orderlyL1BridgeAddress = stdJson.readAddress(_json, "$.orderlyL1BridgeAddress");
        modeL1BridgeAddress = stdJson.readAddress(_json, "$.modeL1BridgeAddress");
        lyraL1BridgeAddress = stdJson.readAddress(_json, "$.lyraL1BridgeAddress");
        dripVersion = stdJson.readUint(_json, "$.dripVersion");
        previousDripVersion = stdJson.readUint(_json, "$.previousDripVersion");
        smallOpChainFaucetDripValue = stdJson.readUint(_json, "$.smallOpChainFaucetDripValue");
        smallOpChainFaucetDripInterval = stdJson.readUint(_json, "$.smallOpChainFaucetDripInterval");
        largeOpChainFaucetDripValue = stdJson.readUint(_json, "$.largeOpChainFaucetDripValue");
        largeOpChainFaucetDripInterval = stdJson.readUint(_json, "$.largeOpChainFaucetDripInterval");
        opChainAdminWalletDripValue = stdJson.readUint(_json, "$.opChainAdminWalletDripValue");
        opChainAdminWalletDripInterval = stdJson.readUint(_json, "$.opChainAdminWalletDripInterval");
        largeFaucetsL1BridgeAddresses[0] = opL1BridgeAddress;
        largeFaucetsL1BridgeAddresses[1] = baseL1BridgeAddress;
        smallFaucetsL1BridgeAddresses[0] = zoraL1BridgeAddress;
        smallFaucetsL1BridgeAddresses[1] = pgnL1BridgeAddress;
        smallFaucetsL1BridgeAddresses[2] = orderlyL1BridgeAddress;
        smallFaucetsL1BridgeAddresses[3] = modeL1BridgeAddress;
        smallFaucetsL1BridgeAddresses[4] = lyraL1BridgeAddress;
    }

    function getSmallFaucetsL1BridgeAddressesCount() public view returns (uint256 count) {
        return smallFaucetsL1BridgeAddresses.length;
    }

    function getLargeFaucetsL1BridgeAddressesCount() public view returns (uint256 count) {
        return largeFaucetsL1BridgeAddresses.length;
    }
}
