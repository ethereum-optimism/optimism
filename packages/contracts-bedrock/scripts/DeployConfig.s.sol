// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { Executables } from "scripts/Executables.sol";
import { Chains } from "scripts/Chains.sol";

// Global constant for the `useFaultProofs` slot in the DeployConfig contract, which can be overridden in the testing
// environment.
bytes32 constant USE_FAULT_PROOFS_SLOT = bytes32(uint256(64));

/// @title DeployConfig
/// @notice Represents the configuration required to deploy the system. It is expected
///         to read the file from JSON. A future improvement would be to have fallback
///         values if they are not defined in the JSON themselves.
contract DeployConfig is Script {
    string internal _json;

    address public finalSystemOwner;
    address public superchainConfigGuardian;
    uint256 public l1ChainID;
    uint256 public l2ChainID;
    uint256 public l2BlockTime;
    uint256 public maxSequencerDrift;
    uint256 public sequencerWindowSize;
    uint256 public channelTimeout;
    address public p2pSequencerAddress;
    address public batchInboxAddress;
    address public batchSenderAddress;
    uint256 public l2OutputOracleSubmissionInterval;
    int256 internal _l2OutputOracleStartingTimestamp;
    uint256 public l2OutputOracleStartingBlockNumber;
    address public l2OutputOracleProposer;
    address public l2OutputOracleChallenger;
    uint256 public finalizationPeriodSeconds;
    bool public fundDevAccounts;
    address public proxyAdminOwner;
    address public baseFeeVaultRecipient;
    uint256 public baseFeeVaultMinimumWithdrawalAmount;
    address public l1FeeVaultRecipient;
    uint256 public l1FeeVaultMinimumWithdrawalAmount;
    address public sequencerFeeVaultRecipient;
    uint256 public sequencerFeeVaultMinimumWithdrawalAmount;
    uint256 public sequencerFeeVaultWithdrawalNetwork;
    string public governanceTokenName;
    string public governanceTokenSymbol;
    address public governanceTokenOwner;
    uint256 public l2GenesisBlockGasLimit;
    uint256 public l2GenesisBlockBaseFeePerGas;
    uint256 public gasPriceOracleOverhead;
    uint256 public gasPriceOracleScalar;
    bool public enableGovernance;
    uint256 public eip1559Denominator;
    uint256 public eip1559Elasticity;
    uint256 public faultGameAbsolutePrestate;
    uint256 public faultGameGenesisBlock;
    bytes32 public faultGameGenesisOutputRoot;
    uint256 public faultGameMaxDepth;
    uint256 public faultGameSplitDepth;
    uint256 public faultGameMaxDuration;
    uint256 public faultGameWithdrawalDelay;
    uint256 public preimageOracleMinProposalSize;
    uint256 public preimageOracleChallengePeriod;
    uint256 public preimageOracleCancunActivationTimestamp;
    uint256 public systemConfigStartBlock;
    uint256 public requiredProtocolVersion;
    uint256 public recommendedProtocolVersion;
    uint256 public proofMaturityDelaySeconds;
    uint256 public disputeGameFinalityDelaySeconds;
    uint256 public respectedGameType;
    bool public useFaultProofs;
    bool public usePlasma;
    uint256 public daChallengeWindow;
    uint256 public daResolveWindow;
    uint256 public daBondSize;
    uint256 public daResolverRefundPercentage;

    function read(string memory _path) public {
        console.log("DeployConfig: reading file %s", _path);
        try vm.readFile(_path) returns (string memory data) {
            _json = data;
        } catch {
            require(false, string.concat("Cannot find deploy config file at ", _path));
        }

        finalSystemOwner = stdJson.readAddress(_json, "$.finalSystemOwner");
        superchainConfigGuardian = stdJson.readAddress(_json, "$.superchainConfigGuardian");
        l1ChainID = stdJson.readUint(_json, "$.l1ChainID");
        l2ChainID = stdJson.readUint(_json, "$.l2ChainID");
        l2BlockTime = stdJson.readUint(_json, "$.l2BlockTime");
        maxSequencerDrift = stdJson.readUint(_json, "$.maxSequencerDrift");
        sequencerWindowSize = stdJson.readUint(_json, "$.sequencerWindowSize");
        channelTimeout = stdJson.readUint(_json, "$.channelTimeout");
        p2pSequencerAddress = stdJson.readAddress(_json, "$.p2pSequencerAddress");
        batchInboxAddress = stdJson.readAddress(_json, "$.batchInboxAddress");
        batchSenderAddress = stdJson.readAddress(_json, "$.batchSenderAddress");
        l2OutputOracleSubmissionInterval = stdJson.readUint(_json, "$.l2OutputOracleSubmissionInterval");
        _l2OutputOracleStartingTimestamp = stdJson.readInt(_json, "$.l2OutputOracleStartingTimestamp");
        l2OutputOracleStartingBlockNumber = stdJson.readUint(_json, "$.l2OutputOracleStartingBlockNumber");
        l2OutputOracleProposer = stdJson.readAddress(_json, "$.l2OutputOracleProposer");
        l2OutputOracleChallenger = stdJson.readAddress(_json, "$.l2OutputOracleChallenger");
        finalizationPeriodSeconds = stdJson.readUint(_json, "$.finalizationPeriodSeconds");
        fundDevAccounts = stdJson.readBool(_json, "$.fundDevAccounts");
        proxyAdminOwner = stdJson.readAddress(_json, "$.proxyAdminOwner");
        baseFeeVaultRecipient = stdJson.readAddress(_json, "$.baseFeeVaultRecipient");
        baseFeeVaultMinimumWithdrawalAmount = stdJson.readUint(_json, "$.baseFeeVaultMinimumWithdrawalAmount");
        l1FeeVaultRecipient = stdJson.readAddress(_json, "$.l1FeeVaultRecipient");
        l1FeeVaultMinimumWithdrawalAmount = stdJson.readUint(_json, "$.l1FeeVaultMinimumWithdrawalAmount");
        sequencerFeeVaultRecipient = stdJson.readAddress(_json, "$.sequencerFeeVaultRecipient");
        sequencerFeeVaultMinimumWithdrawalAmount = stdJson.readUint(_json, "$.sequencerFeeVaultMinimumWithdrawalAmount");
        sequencerFeeVaultWithdrawalNetwork = stdJson.readUint(_json, "$.sequencerFeeVaultWithdrawalNetwork");
        governanceTokenName = stdJson.readString(_json, "$.governanceTokenName");
        governanceTokenSymbol = stdJson.readString(_json, "$.governanceTokenSymbol");
        governanceTokenOwner = stdJson.readAddress(_json, "$.governanceTokenOwner");
        l2GenesisBlockGasLimit = stdJson.readUint(_json, "$.l2GenesisBlockGasLimit");
        l2GenesisBlockBaseFeePerGas = stdJson.readUint(_json, "$.l2GenesisBlockBaseFeePerGas");
        gasPriceOracleOverhead = stdJson.readUint(_json, "$.gasPriceOracleOverhead");
        gasPriceOracleScalar = stdJson.readUint(_json, "$.gasPriceOracleScalar");
        enableGovernance = stdJson.readBool(_json, "$.enableGovernance");
        eip1559Denominator = stdJson.readUint(_json, "$.eip1559Denominator");
        eip1559Elasticity = stdJson.readUint(_json, "$.eip1559Elasticity");
        systemConfigStartBlock = stdJson.readUint(_json, "$.systemConfigStartBlock");
        requiredProtocolVersion = stdJson.readUint(_json, "$.requiredProtocolVersion");
        recommendedProtocolVersion = stdJson.readUint(_json, "$.recommendedProtocolVersion");

        useFaultProofs = stdJson.readBool(_json, "$.useFaultProofs");
        proofMaturityDelaySeconds = stdJson.readUint(_json, "$.proofMaturityDelaySeconds");
        disputeGameFinalityDelaySeconds = stdJson.readUint(_json, "$.disputeGameFinalityDelaySeconds");
        respectedGameType = stdJson.readUint(_json, "$.respectedGameType");

        faultGameAbsolutePrestate = stdJson.readUint(_json, "$.faultGameAbsolutePrestate");
        faultGameMaxDepth = stdJson.readUint(_json, "$.faultGameMaxDepth");
        faultGameSplitDepth = stdJson.readUint(_json, "$.faultGameSplitDepth");
        faultGameMaxDuration = stdJson.readUint(_json, "$.faultGameMaxDuration");
        faultGameGenesisBlock = stdJson.readUint(_json, "$.faultGameGenesisBlock");
        faultGameGenesisOutputRoot = stdJson.readBytes32(_json, "$.faultGameGenesisOutputRoot");
        faultGameWithdrawalDelay = stdJson.readUint(_json, "$.faultGameWithdrawalDelay");

        preimageOracleMinProposalSize = stdJson.readUint(_json, "$.preimageOracleMinProposalSize");
        preimageOracleChallengePeriod = stdJson.readUint(_json, "$.preimageOracleChallengePeriod");
        preimageOracleCancunActivationTimestamp = stdJson.readUint(_json, "$.preimageOracleCancunActivationTimestamp");

        usePlasma = _readOr(_json, "$.usePlasma", false);
        daChallengeWindow = _readOr(_json, "$.daChallengeWindow", 1000);
        daResolveWindow = _readOr(_json, "$.daResolveWindow", 1000);
        daBondSize = _readOr(_json, "$.daBondSize", 1000000000);
        daResolverRefundPercentage = _readOr(_json, "$.daResolverRefundPercentage", 0);
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

    /// @notice Allow the `usePlasma` config to be overridden in testing environments
    function setUsePlasma(bool _usePlasma) public {
        usePlasma = _usePlasma;
    }

    function _getBlockByTag(string memory _tag) internal returns (bytes32) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat("cast block ", _tag, " --json | ", Executables.jq, " -r .hash");
        bytes memory res = vm.ffi(cmd);
        return abi.decode(res, (bytes32));
    }

    function _readOr(string memory json, string memory key, bool defaultValue) internal view returns (bool) {
        return vm.keyExists(json, key) ? stdJson.readBool(json, key) : defaultValue;
    }

    function _readOr(string memory json, string memory key, uint256 defaultValue) internal view returns (uint256) {
        return vm.keyExists(json, key) ? stdJson.readUint(json, key) : defaultValue;
    }
}
