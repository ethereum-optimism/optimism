// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { Executables } from "./Executables.sol";
import { Chains } from "./Chains.sol";

/// @title DeployConfig
/// @notice Represents the configuration required to deploy the system. It is expected
///         to read the file from JSON. Values are accessed lazily via getter funtions.
///         Need to use crazy hacks with assembly to cast function pointers to view.
contract DeployConfig is Script {
    string internal _json;

    constructor(string memory _path) {
        console.log("DeployConfig: reading file %s", _path);
        try vm.readFile(_path) returns (string memory data) {
            _json = data;
        } catch {
            console.log("Warning: unable to read config. Do not deploy unless you are not using config.");
        }
    }

    function _castViewUint256(function () returns (uint256) f) internal view returns (uint256) {
        function() view returns (uint256) inner;
        function() returns (uint256) i = f;
        assembly {
            inner := i
        }
        return inner();
    }

    function _castViewAddress(function () returns (address) f) internal view returns (address) {
        function() view returns (address) inner;
        function() returns (address) i = f;
        assembly {
            inner := i
        }
        return inner();
    }

    function _castViewString(function () returns (string memory) f) internal view returns (string memory) {
        function() view returns (string memory) inner;
        function() returns (string memory) i = f;
        assembly {
            inner := i
        }
        return inner();
    }

    function finalSystemOwner() public view returns (address) {
        return _castViewAddress(_finalSystemOwner);
    }

    function _finalSystemOwner() internal returns (address) {
        return stdJson.readAddress(_json, "$.finalSystemOwner");
    }

    function portalGuardian() public view returns (address) {
        return _castViewAddress(_portalGuardian);
    }

    function _portalGuardian() internal returns (address) {
        return stdJson.readAddress(_json, "$.portalGuardian");
    }

    function l1ChainID() public view returns (uint256) {
        return _castViewUint256(_l1ChainID);
    }

    function _l1ChainID() internal returns (uint256) {
        return stdJson.readUint(_json, "$.l1ChainID");
    }

    function l2ChainID() public view returns (uint256) {
        return _castViewUint256(_l2ChainID);
    }

    function _l2ChainID() internal returns (uint256) {
        return stdJson.readUint(_json, "$.l2ChainID");
    }

    function l2BlockTime() public view returns (uint256) {
        return _castViewUint256(_l2BlockTime);
    }

    function _l2BlockTime() internal returns (uint256) {
        return stdJson.readUint(_json, "$.l2BlockTime");
    }

    function maxSequencerDrift() public view returns (uint256) {
        return _castViewUint256(_maxSequencerDrift);
    }

    function _maxSequencerDrift() internal returns (uint256) {
        return stdJson.readUint(_json, "$.maxSequencerDrift");
    }

    function sequencerWindowSize() public view returns (uint256) {
        return _castViewUint256(_sequencerWindowSize);
    }

    function _sequencerWindowSize() internal returns (uint256) {
        return stdJson.readUint(_json, "$.sequencerWindowSize");
    }

    function channelTimeout() public view returns (uint256) {
        return _castViewUint256(_channelTimeout);
    }

    function _channelTimeout() internal returns (uint256) {
        return stdJson.readUint(_json, "$.channelTimeout");
    }

    function p2pSequencerAddress() public view returns (address) {
        return _castViewAddress(_p2pSequencerAddress);
    }

    function _p2pSequencerAddress() internal returns (address) {
        return stdJson.readAddress(_json, "$.p2pSequencerAddress");
    }

    function batchInboxAddress() public view returns (address) {
        return _castViewAddress(_batchInboxAddress);
    }

    function _batchInboxAddress() internal returns (address) {
        return stdJson.readAddress(_json, "$.batchInboxAddress");
    }

    function batchSenderAddress() public view returns (address) {
        return _castViewAddress(_batchSenderAddress);
    }

    function _batchSenderAddress() internal returns (address) {
        return stdJson.readAddress(_json, "$.batchSenderAddress");
    }

    function l2OutputOracleSubmissionInterval() public view returns (uint256) {
        return _castViewUint256(_l2OutputOracleSubmissionInterval);
    }

    function _l2OutputOracleSubmissionInterval() internal returns (uint256) {
        return stdJson.readUint(_json, "$.l2OutputOracleSubmissionInterval");
    }

    function l2OutputOracleStartingBlockNumber() public view returns (uint256) {
        return _castViewUint256(_l2OutputOracleStartingBlockNumber);
    }

    function _l2OutputOracleStartingBlockNumber() internal returns (uint256) {
        return stdJson.readUint(_json, "$.l2OutputOracleStartingBlockNumber");
    }

    function l2OutputOracleProposer() public view returns (address) {
        return _castViewAddress(_l2OutputOracleProposer);
    }

    function _l2OutputOracleProposer() internal returns (address) {
        return stdJson.readAddress(_json, "$.l2OutputOracleProposer");
    }

    function l2OutputOracleChallenger() public view returns (address) {
        return _castViewAddress(_l2OutputOracleChallenger);
    }

    function _l2OutputOracleChallenger() internal returns (address) {
        return stdJson.readAddress(_json, "$.l2OutputOracleChallenger");
    }

    function finalizationPeriodSeconds() public view returns (uint256) {
        return _castViewUint256(_finalizationPeriodSeconds);
    }

    function _finalizationPeriodSeconds() internal returns (uint256) {
        return stdJson.readUint(_json, "$.finalizationPeriodSeconds");
    }

    function proxyAdminOwner() public view returns (address) {
        return _castViewAddress(_proxyAdminOwner);
    }

    function _proxyAdminOwner() internal returns (address) {
        return stdJson.readAddress(_json, "$.proxyAdminOwner");
    }

    function baseFeeVaultRecipient() public view returns (address) {
        return _castViewAddress(_baseFeeVaultRecipient);
    }

    function _baseFeeVaultRecipient() internal returns (address) {
        return stdJson.readAddress(_json, "$.baseFeeVaultRecipient");
    }

    function l1FeeVaultRecipient() public view returns (address) {
        return _castViewAddress(_l1FeeVaultRecipient);
    }

    function _l1FeeVaultRecipient() internal returns (address) {
        return stdJson.readAddress(_json, "$.l1FeeVaultRecipient");
    }

    function sequencerFeeVaultRecipient() public view returns (address) {
        return _castViewAddress(_sequencerFeeVaultRecipient);
    }

    function _sequencerFeeVaultRecipient() internal returns (address) {
        return stdJson.readAddress(_json, "$.sequencerFeeVaultRecipient");
    }

    function governanceTokenName() public view returns (string memory) {
        return _castViewString(_governanceTokenName);
    }

    function _governanceTokenName() internal returns (string memory) {
        return stdJson.readString(_json, "$.governanceTokenName");
    }

    function governanceTokenSymbol() public view returns (string memory) {
        return _castViewString(_governanceTokenSymbol);
    }

    function _governanceTokenSymbol() internal returns (string memory) {
        return stdJson.readString(_json, "$.governanceTokenSymbol");
    }

    function governanceTokenOwner() public view returns (address) {
        return _castViewAddress(_governanceTokenOwner);
    }

    function _governanceTokenOwner() internal returns (address) {
        return stdJson.readAddress(_json, "$.governanceTokenOwner");
    }

    function l2GenesisBlockGasLimit() public view returns (uint256) {
        return _castViewUint256(_l2GenesisBlockGasLimit);
    }

    function _l2GenesisBlockGasLimit() internal returns (uint256) {
        return stdJson.readUint(_json, "$.l2GenesisBlockGasLimit");
    }

    function l2GenesisBlockBaseFeePerGas() public view returns (uint256) {
        return _castViewUint256(_l2GenesisBlockBaseFeePerGas);
    }

    function _l2GenesisBlockBaseFeePerGas() internal returns (uint256) {
        return stdJson.readUint(_json, "$.l2GenesisBlockBaseFeePerGas");
    }

    function gasPriceOracleOverhead() public view returns (uint256) {
        return _castViewUint256(_gasPriceOracleOverhead);
    }

    function _gasPriceOracleOverhead() internal returns (uint256) {
        return stdJson.readUint(_json, "$.gasPriceOracleOverhead");
    }

    function gasPriceOracleScalar() public view returns (uint256) {
        return _castViewUint256(_gasPriceOracleScalar);
    }

    function _gasPriceOracleScalar() internal returns (uint256) {
        return stdJson.readUint(_json, "$.gasPriceOracleScalar");
    }

    function eip1559Denominator() public view returns (uint256) {
        return _castViewUint256(_eip1559Denominator);
    }

    function _eip1559Denominator() internal returns (uint256) {
        return stdJson.readUint(_json, "$.eip1559Denominator");
    }

    function eip1559Elasticity() public view returns (uint256) {
        return _castViewUint256(_eip1559Elasticity);
    }

    function _eip1559Elasticity() internal returns (uint256) {
        return stdJson.readUint(_json, "$.eip1559Elasticity");
    }

    function systemConfigStartBlock() public view returns (uint256) {
        return _castViewUint256(_systemConfigStartBlock);
    }

    function _systemConfigStartBlock() internal returns (uint256) {
        return stdJson.readUint(_json, "$.systemConfigStartBlock");
    }

    function requiredProtocolVersion() public view returns (uint256) {
        return _castViewUint256(_requiredProtocolVersion);
    }

    function _requiredProtocolVersion() internal returns (uint256) {
        return stdJson.readUint(_json, "$.requiredProtocolVersion");
    }

    function recommendedProtocolVersion() public view returns (uint256) {
        return _castViewUint256(_recommendedProtocolVersion);
    }

    function _recommendedProtocolVersion() internal returns (uint256) {
        return stdJson.readUint(_json, "$.recommendedProtocolVersion");
    }

    function faultGameAbsolutePrestate() public view returns (uint256) {
        return _castViewUint256(_faultGameAbsolutePrestate);
    }

    function _faultGameAbsolutePrestate() internal returns (uint256) {
        return stdJson.readUint(_json, "$.faultGameAbsolutePrestate");
    }

    function faultGameMaxDepth() public view returns (uint256) {
        return _castViewUint256(_faultGameMaxDepth);
    }

    function _faultGameMaxDepth() internal returns (uint256) {
        return stdJson.readUint(_json, "$.faultGameMaxDepth");
    }

    function faultGameMaxDuration() public view returns (uint256) {
        return _castViewUint256(_faultGameMaxDuration);
    }

    function _faultGameMaxDuration() public returns (uint256) {
        return stdJson.readUint(_json, "$.faultGameMaxDuration");
    }

    function l1StartingBlockTag() public returns (bytes32) {
        try vm.parseJsonBytes32(_json, "$.l1StartingBlockTag") returns (bytes32 tag) {
            return tag;
        } catch {
            try vm.parseJsonString(_json, "$.l1StartingBlockTag") returns (string memory tag) {
                return _getBlockByTag(tag);
            } catch {
                try vm.parseJsonUint(_json, "$.l1StartingBlockTag") returns (uint256 tag) {
                    return _getBlockByTag(vm.toString(tag));
                } catch { }
            }
        }
        revert("l1StartingBlockTag must be a bytes32, string or uint256 or cannot fetch l1StartingBlockTag");
    }

    function l2OutputOracleStartingTimestamp() public returns (uint256) {
        int256 _l2OutputOracleStartingTimestamp = stdJson.readInt(_json, "$.l2OutputOracleStartingTimestamp");
        if (_l2OutputOracleStartingTimestamp < 0) {
            bytes32 tag = l1StartingBlockTag();
            string[] memory cmd = new string[](3);
            cmd[0] = Executables.bash;
            cmd[1] = "-c";
            cmd[2] = string.concat("cast block ", vm.toString(tag), " --json | ", Executables.jq, " .timestamp");
            bytes memory res = vm.ffi(cmd);
            return stdJson.readUint(string(res), "");
        }
        return uint256(_l2OutputOracleStartingTimestamp);
    }

    function _getBlockByTag(string memory _tag) internal returns (bytes32) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat("cast block ", _tag, " --json | ", Executables.jq, " -r .hash");
        bytes memory res = vm.ffi(cmd);
        return abi.decode(res, (bytes32));
    }
}
