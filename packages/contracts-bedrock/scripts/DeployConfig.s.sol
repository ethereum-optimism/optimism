// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { Executables } from "scripts/Executables.sol";
import { Chains } from "scripts/Chains.sol";

// Global constant for the `useFaultProofs` slot in the DeployConfig contract, which can be overridden in the testing
// environment.
bytes32 constant USE_FAULT_PROOFS_SLOT = bytes32(uint256(63));

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
    uint256 public baseFeeVaultWithdrawalNetwork;
    address public l1FeeVaultRecipient;
    uint256 public l1FeeVaultMinimumWithdrawalAmount;
    uint256 public l1FeeVaultWithdrawalNetwork;
    address public sequencerFeeVaultRecipient;
    uint256 public sequencerFeeVaultMinimumWithdrawalAmount;
    uint256 public sequencerFeeVaultWithdrawalNetwork;
    string public governanceTokenName;
    string public governanceTokenSymbol;
    address public governanceTokenOwner;
    uint256 public l2GenesisBlockNumber;
    uint256 public l2GenesisBlockGasLimit;
    uint256 public l2GenesisBlockBaseFeePerGas;
    uint256 public gasPriceOracleOverhead;
    uint256 public gasPriceOracleScalar;
    bool public enableGovernance;
    uint256 public faultGameAbsolutePrestate;
    uint256 public faultGameGenesisBlock;
    bytes32 public faultGameGenesisOutputRoot;
    uint256 public faultGameMaxDepth;
    uint256 public faultGameSplitDepth;
    uint256 public faultGameMaxDuration;
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

    //////////////////////////////////////////////////////
    /// Genesis Block Properties
    //////////////////////////////////////////////////////

    uint256 public bedrockBlock;

    int256 public l2GenesisRegolithTimeOffset;
    int256 public l2GenesisCanyonTimeOffset;
    int256 public l2GenesisEcotoneTimeOffset;
    int256 public l2GenesisInteropTimeOffset;

    uint256 public eip1559Elasticity;
    uint256 public eip1559Denominator;
    uint256 public eip1559DenominatorCanyon;

    uint256 public nonce;
    uint256 public timestamp;
    bytes public extraData;
    uint256 public gasLimit;
    uint256 public difficulty;
    bytes public mixHash;
    address public coinbase;
    uint256 public number;
    uint256 public gasUsed;
    bytes public parentHash;
    uint256 public baseFeePerGas;

    function read(string memory _path) public {
        console.log("DeployConfig: reading file %s", _path);
        try vm.readFile(_path) returns (string memory data) {
            _json = data;
        } catch {
            console.log("Warning: unable to read config. Do not deploy unless you are not using config.");
            return;
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
        baseFeeVaultWithdrawalNetwork = stdJson.readUint(_json, "$.baseFeeVaultWithdrawalNetwork");
        l1FeeVaultRecipient = stdJson.readAddress(_json, "$.l1FeeVaultRecipient");
        l1FeeVaultMinimumWithdrawalAmount = stdJson.readUint(_json, "$.l1FeeVaultMinimumWithdrawalAmount");
        l1FeeVaultWithdrawalNetwork = stdJson.readUint(_json, "$.l1FeeVaultWithdrawalNetwork");
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

        preimageOracleMinProposalSize = stdJson.readUint(_json, "$.preimageOracleMinProposalSize");
        preimageOracleChallengePeriod = stdJson.readUint(_json, "$.preimageOracleChallengePeriod");
        preimageOracleCancunActivationTimestamp = stdJson.readUint(_json, "$.preimageOracleCancunActivationTimestamp");

        //////////////////////////////////////////////////////
        /// Genesis Config Properties
        //////////////////////////////////////////////////////
        l2GenesisBlockNumber = stdJson.readUint(_json, "$.l2GenesisBlockNumber");
        bedrockBlock = l2GenesisBlockNumber;

        l2GenesisRegolithTimeOffset = parseJsonIntWithDefault("$.l2GenesisRegolithTimeOffset", int256(-1));
        l2GenesisCanyonTimeOffset = parseJsonIntWithDefault("$.l2GenesisCanyonTimeOffset", int256(-1));
        l2GenesisEcotoneTimeOffset = parseJsonIntWithDefault("$.l2GenesisEcotoneTimeOffset", int256(-1));
        l2GenesisInteropTimeOffset = parseJsonIntWithDefault("$.l2GenesisInteropTimeOffset", int256(-1));

        eip1559Elasticity = parseJsonUintWithDefault("$.eip1559Elasticity", uint256(10));
        eip1559Denominator = parseJsonUintWithDefault("$.eip1559Denominator", uint256(50));
        eip1559DenominatorCanyon = parseJsonUintWithDefault("$.eip1559DenominatorCanyon", uint256(250));

        nonce = parseJsonUintWithDefault("$.l2GenesisBlockNonce", uint256(0));
        // TODO block.timestamp
        // timestamp = parseJsonUintWithDefault("$.timestamp", block.timestamp);
        /// @notice 424544524f434b == BEDROCK
        extraData = parseJsonBytesWithDefault("$.extraData", hex"424544524f434b");
        gasLimit = parseJsonUintWithDefault("$.gasLimit", uint256(30_000_000));
        // TODO difficulty.ToInt() in Go code
        // difficulty = parseJsonUintWithDefault("$.difficulty", uint256(0));
        mixHash = parseJsonBytesWithDefault(
            "$.l2GenesisBlockMixHash", hex"0000000000000000000000000000000000000000000000000000000000000000"
        );
        // TODO 0x4200000000000000000000000000000000000011
        // try vm.parseJsonAddress(_json, "$.coinbase") returns (address _data) {
        //     coinbase = _data;
        // } catch {
        //     coinbase = 0x4200000000000000000000000000000000000011;
        // }
        number = l2GenesisBlockNumber;
        // TODO L2GenesisBlockGasUsed
        // gasUsed = parseJsonUintWithDefault("$.gasUsed", uint256(0));
        // TODO L2GenesisBlockParentHash
        // parentHash = parseJsonBytesWithDefault("$.parentHash",
        // hex'0000000000000000000000000000000000000000000000000000000000000000');
        baseFeePerGas = l2GenesisBlockBaseFeePerGas;
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

    function _getBlockByTag(string memory _tag) internal returns (bytes32) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat("cast block ", _tag, " --json | ", Executables.jq, " -r .hash");
        bytes memory res = vm.ffi(cmd);
        return abi.decode(res, (bytes32));
    }

    function parseJsonUintWithDefault(string memory _path, uint256 _defaultValue) internal view returns (uint256) {
        try vm.parseJsonUint(_json, _path) returns (uint256 _data) {
            return _data;
        } catch {
            return _defaultValue;
        }
    }

    function parseJsonIntWithDefault(string memory _path, int256 _defaultValue) internal view returns (int256) {
        try vm.parseJsonInt(_json, _path) returns (int256 _data) {
            return _data;
        } catch {
            return _defaultValue;
        }
    }

    function parseJsonBytesWithDefault(
        string memory _path,
        bytes memory _defaultValue
    )
        internal
        view
        returns (bytes memory)
    {
        try vm.parseJsonBytes(_json, _path) returns (bytes memory _data) {
            return _data;
        } catch {
            return _defaultValue;
        }
    }
}
