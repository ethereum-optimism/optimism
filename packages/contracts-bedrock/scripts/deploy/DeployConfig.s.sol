// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { Executables } from "scripts/Executables.sol";
import { Process } from "scripts/libraries/Process.sol";
import { Chains } from "scripts/Chains.sol";
import { Config, Fork, ForkUtils } from "scripts/Config.sol";

/// @title DeployConfig
/// @notice Represents the configuration required to deploy the system. It is expected
///         to read the file from JSON. A future improvement would be to have fallback
///         values if they are not defined in the JSON themselves.
contract DeployConfig is Script {
    using stdJson for string;
    using ForkUtils for Fork;

    /// @notice Represents an unset offset value, as opposed to 0, which denotes no-offset.
    uint256 constant NULL_OFFSET = type(uint256).max;

    string internal _json;

    function basefeeScalar() public returns (uint32 out_) {
        return uint32(_readOr(_json, "$.gasPriceOracleBaseFeeScalar", 1368));
    }

    function baseFeeVaultMinimumWithdrawalAmount() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.baseFeeVaultMinimumWithdrawalAmount");
    }

    function baseFeeVaultRecipient() public returns (address out_) {
        return stdJson.readAddress(_json, "$.baseFeeVaultRecipient");
    }

    function baseFeeVaultWithdrawalNetwork() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.baseFeeVaultWithdrawalNetwork");
    }

    function batchInboxAddress() public returns (address out_) {
        return stdJson.readAddress(_json, "$.batchInboxAddress");
    }

    function batchSenderAddress() public returns (address out_) {
        return stdJson.readAddress(_json, "$.batchSenderAddress");
    }

    function blobbasefeeScalar() public returns (uint32 out_) {
        return uint32(_readOr(_json, "$.gasPriceOracleBlobBaseFeeScalar", 810949));
    }

    function channelTimeout() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.channelTimeout");
    }

    function customGasTokenAddress() public returns (address out_) {
        if (overrideUseInteropSet) return overrideCustomGasTokenAddress;
        return _readOr(_json, "$.customGasTokenAddress", address(0));
    }

    function daBondSize() public returns (uint256 out_) {
        return _readOr(_json, "$.daBondSize", 1000000000);
    }

    function daChallengeWindow() public returns (uint256 out_) {
        return _readOr(_json, "$.daChallengeWindow", 1000);
    }

    function daCommitmentType() public returns (string out_) {
        return _readOr(_json, "$.daCommitmentType", "KeccakCommitment");
    }

    function daResolverRefundPercentage() public returns (uint256 out_) {
        return _readOr(_json, "$.daResolverRefundPercentage", 0);
    }

    function daResolveWindow() public returns (uint256 out_) {
        return _readOr(_json, "$.daResolveWindow", 1000);
    }

    function disputeGameFinalityDelaySeconds() public returns (uint256 out_) {
        return _readOr(_json, "$.disputeGameFinalityDelaySeconds", 0);
    }

    function eip1559Denominator() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.eip1559Denominator");
    }

    function eip1559Elasticity() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.eip1559Elasticity");
    }

    function enableGovernance() public returns (bool out_) {
        return stdJson.readBool(_json, "$.enableGovernance");
    }

    function faultGameAbsolutePrestate() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.faultGameAbsolutePrestate");
    }

    function faultGameClockExtension() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.faultGameClockExtension");
    }

    function faultGameGenesisBlock() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.faultGameGenesisBlock");
    }

    function faultGameGenesisOutputRoot() public returns (bytes32 out_) {
        return stdJson.readBytes32(_json, "$.faultGameGenesisOutputRoot");
    }

    function faultGameMaxClockDuration() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.faultGameMaxClockDuration");
    }

    function faultGameMaxDepth() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.faultGameMaxDepth");
    }

    function faultGameSplitDepth() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.faultGameSplitDepth");
    }

    function faultGameWithdrawalDelay() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.faultGameWithdrawalDelay");
    }

    function finalizationPeriodSeconds() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.finalizationPeriodSeconds");
    }

    function finalSystemOwner() public returns (address out_) {
        return stdJson.readAddress(_json, "$.finalSystemOwner");
    }

    bool internal overrideFundDevAccountsSet;
    bool internal overrideFundDevAccountsValue;

    function fundDevAccounts() public returns (bool out_) {
        if (overrideFundDevAccountsSet) return overrideFundDevAccountsValue;
        return _readOr(_json, "$.fundDevAccounts", false);
    }

    function governanceTokenName() public returns (string out_) {
        return stdJson.readString(_json, "$.governanceTokenName");
    }

    function governanceTokenOwner() public returns (address out_) {
        return stdJson.readAddress(_json, "$.governanceTokenOwner");
    }

    function governanceTokenSymbol() public returns (string out_) {
        return stdJson.readString(_json, "$.governanceTokenSymbol");
    }

    function l1ChainID() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.l1ChainID");
    }

    function l1FeeVaultMinimumWithdrawalAmount() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.l1FeeVaultMinimumWithdrawalAmount");
    }

    function l1FeeVaultRecipient() public returns (address out_) {
        return stdJson.readAddress(_json, "$.l1FeeVaultRecipient");
    }

    function l1FeeVaultWithdrawalNetwork() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.l1FeeVaultWithdrawalNetwork");
    }

    function l2BlockTime() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.l2BlockTime");
    }

    function l2ChainID() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.l2ChainID");
    }

    function l2GenesisBlockGasLimit() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.l2GenesisBlockGasLimit");
    }

    function l2GenesisDeltaTimeOffset() public returns (uint256 out_) {
        return _readOr(_json, "$.l2GenesisDeltaTimeOffset", NULL_OFFSET);
    }

    function l2GenesisEcotoneTimeOffset() public returns (uint256 out_) {
        return _readOr(_json, "$.l2GenesisEcotoneTimeOffset", NULL_OFFSET);
    }

    function l2GenesisFjordTimeOffset() public returns (uint256 out_) {
        return _readOr(_json, "$.l2GenesisFjordTimeOffset", NULL_OFFSET);
    }

    function l2OutputOracleChallenger() public returns (address out_) {
        return stdJson.readAddress(_json, "$.l2OutputOracleChallenger");
    }

    function l2OutputOracleProposer() public returns (address out_) {
        return stdJson.readAddress(_json, "$.l2OutputOracleProposer");
    }

    function l2OutputOracleStartingBlockNumber() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.l2OutputOracleStartingBlockNumber");
    }

    function _l2OutputOracleStartingTimestamp() internal returns (int256 out_) {
        return stdJson.readInt(_json, "$.l2OutputOracleStartingTimestamp");
    }

    function l2OutputOracleSubmissionInterval() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.l2OutputOracleSubmissionInterval");
    }

    function maxSequencerDrift() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.maxSequencerDrift");
    }

    function p2pSequencerAddress() public returns (address out_) {
        return stdJson.readAddress(_json, "$.p2pSequencerAddress");
    }

    function preimageOracleChallengePeriod() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.preimageOracleChallengePeriod");
    }

    function preimageOracleMinProposalSize() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.preimageOracleMinProposalSize");
    }

    function proofMaturityDelaySeconds() public returns (uint256 out_) {
        return _readOr(_json, "$.proofMaturityDelaySeconds", 0);
    }

    function proxyAdminOwner() public returns (address out_) {
        return stdJson.readAddress(_json, "$.proxyAdminOwner");
    }

    function recommendedProtocolVersion() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.recommendedProtocolVersion");
    }

    function requiredProtocolVersion() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.requiredProtocolVersion");
    }

    function respectedGameType() public returns (uint256 out_) {
        return _readOr(_json, "$.respectedGameType", 0);
    }

    function sequencerFeeVaultMinimumWithdrawalAmount() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.sequencerFeeVaultMinimumWithdrawalAmount");
    }

    function sequencerFeeVaultRecipient() public returns (address out_) {
        return stdJson.readAddress(_json, "$.sequencerFeeVaultRecipient");
    }

    function sequencerFeeVaultWithdrawalNetwork() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.sequencerFeeVaultWithdrawalNetwork");
    }

    function sequencerWindowSize() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.sequencerWindowSize");
    }

    function superchainConfigGuardian() public returns (address out_) {
        return stdJson.readAddress(_json, "$.superchainConfigGuardian");
    }

    function systemConfigStartBlock() public returns (uint256 out_) {
        return stdJson.readUint(_json, "$.systemConfigStartBlock");
    }

    bool internal overrideUseCustomGasTokenSet;
    bool internal overrideUseCustomGasTokenValue;
    address internal overrideCustomGasTokenAddress;

    function useCustomGasToken() public returns (bool out_) {
        if (overrideUseCustomGasTokenSet) return overrideUseCustomGasTokenValue;
        return _readOr(_json, "$.useCustomGasToken", false);
    }

    bool internal overrideUseFaultProofsSet;
    bool internal overrideUseFaultProofsValue;

    function useFaultProofs() public returns (bool out_) {
        if (overrideUseFaultProofsSet) return overrideUseFaultProofsValue;
        return _readOr(_json, "$.useFaultProofs", false);
    }

    bool internal overrideUseInteropSet;
    bool internal overrideUseInteropValue;

    function useInterop() public returns (bool out_) {
        if (overrideUseInteropSet) return overrideUseInteropValue;
        return _readOr(_json, "$.useInterop", false);
    }

    bool internal overrideUsePlasmaSet;
    bool internal overrideUsePlasmaValue;

    function usePlasma() public returns (bool out_) {
        if (overrideUsePlasmaSet) return overrideUsePlasmaValue;
        return _readOr(_json, "$.usePlasma", false);
    }

    function read(string memory _path) public {
        console.log("DeployConfig: reading file %s", _path);
        try vm.readFile(_path) returns (string memory data) {
            _json = data;
        } catch {
            require(false, string.concat("Cannot find deploy config file at ", _path));
        }
    }

    function fork() public view returns (Fork fork_) {
        // let env var take precedence
        fork_ = Config.fork();
        if (fork_ == Fork.NONE) {
            // Will revert if no deploy config can be found either.
            fork_ = latestGenesisFork();
            console.log("DeployConfig: using deploy config fork: %s", fork_.toString());
        } else {
            console.log("DeployConfig: using env var fork: %s", fork_.toString());
        }
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
        int256 timestamp = _l2OutputOracleStartingTimestamp();
        if (timestamp < 0) {
            bytes32 tag = l1StartingBlockTag();
            string[] memory cmd = new string[](3);
            cmd[0] = Executables.bash;
            cmd[1] = "-c";
            cmd[2] = string.concat("cast block ", vm.toString(tag), " --json | ", Executables.jq, " .timestamp");
            bytes memory res = Process.run(cmd);
            return stdJson.readUint(string(res), "");
        }
        return uint256(timestamp);
    }

    /// @notice Allow the `usePlasma` config to be overridden in testing environments
    function setUsePlasma(bool _usePlasma) public {
        overrideUsePlasmaSet = true;
        overrideUsePlasmaValue = _usePlasma;
    }

    /// @notice Allow the `useFaultProofs` config to be overridden in testing environments
    function setUseFaultProofs(bool _useFaultProofs) public {
        overrideUseFaultProofsSet = true;
        overrideUseFaultProofsValue = _useFaultProofs;
    }

    /// @notice Allow the `useInterop` config to be overridden in testing environments
    function setUseInterop(bool _useInterop) public {
        overrideUseInteropSet = true;
        overrideUseInteropValue = _useInterop;
    }

    /// @notice Allow the `fundDevAccounts` config to be overridden.
    function setFundDevAccounts(bool _fundDevAccounts) public {
        overrideFundDevAccountsSet = true;
        overrideFundDevAccountsValue = _fundDevAccounts;
    }

    /// @notice Allow the `useCustomGasToken` config to be overridden in testing environments
    function setUseCustomGasToken(address _token) public {
        overrideUseCustomGasTokenSet = true;
        overrideUseCustomGasTokenValue = true;
        overrideCustomGasTokenAddress = _token;
    }

    function latestGenesisFork() internal view returns (Fork) {
        if (l2GenesisFjordTimeOffset() == 0) {
            return Fork.FJORD;
        } else if (l2GenesisEcotoneTimeOffset() == 0) {
            return Fork.ECOTONE;
        } else if (l2GenesisDeltaTimeOffset() == 0) {
            return Fork.DELTA;
        }
        revert("DeployConfig: no supported fork active at genesis");
    }

    function _getBlockByTag(string memory _tag) internal returns (bytes32) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat("cast block ", _tag, " --json | ", Executables.jq, " -r .hash");
        bytes memory res = Process.run(cmd);
        return abi.decode(res, (bytes32));
    }

    function _readOr(string memory json, string memory key, bool defaultValue) internal view returns (bool) {
        return vm.keyExistsJson(json, key) ? json.readBool(key) : defaultValue;
    }

    function _readOr(string memory json, string memory key, uint256 defaultValue) internal view returns (uint256) {
        return (vm.keyExistsJson(json, key) && !_isNull(json, key)) ? json.readUint(key) : defaultValue;
    }

    function _readOr(string memory json, string memory key, address defaultValue) internal view returns (address) {
        return vm.keyExistsJson(json, key) ? json.readAddress(key) : defaultValue;
    }

    function _isNull(string memory json, string memory key) internal pure returns (bool) {
        string memory value = json.readString(key);
        return (keccak256(bytes(value)) == keccak256(bytes("null")));
    }

    function _readOr(
        string memory json,
        string memory key,
        string memory defaultValue
    )
        internal
        view
        returns (string memory)
    {
        return vm.keyExists(json, key) ? json.readString(key) : defaultValue;
    }
}
