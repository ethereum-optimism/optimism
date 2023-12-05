// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { Executables } from "scripts/Executables.sol";
import { console2 as console } from "forge-std/console2.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";

/// @title Authorization_Test
/// @dev Specify access requirements of all entrypoints to L1 contracts.
///      When adding new functions, make sure to update the `setUp` function to document if
///      the function should be authorized or not. The `Spec` struct reppresents this
///      documentation, where `auth` is `true` if the function requires authorization and
///      `false` otherwise. However, this contract does not test for authorization, only that
///      an auth spec is defined for every L1 function.
contract Authorization_Test is CommonTest {
    struct AbiEntry {
        string fnName;
        bytes4 sel;
    }

    struct Abi {
        string contractName;
        AbiEntry[] entries;
    }

    struct Spec {
        string name;
        bytes4 sel;
        bool auth;
    }

    mapping(string => mapping(bytes4 => Spec)) specs;
    mapping(string => uint256) public numEntries;

    function setUp() public override {
        super.setUp();

        // DelayedVetoable
        _addSpec("DelayedVetoable", _getSel("delay()"), false);
        _addSpec("DelayedVetoable", _getSel("initiator()"), false);
        _addSpec("DelayedVetoable", _getSel("queuedAt(bytes32)"), false);
        _addSpec("DelayedVetoable", _getSel("target()"), false);
        _addSpec("DelayedVetoable", _getSel("version()"), false);
        _addSpec("DelayedVetoable", _getSel("vetoer()"), false);

        // L1CrossDomainMessenger
        _addSpec("L1CrossDomainMessenger", _getSel("MESSAGE_VERSION()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("MIN_GAS_CALLDATA_OVERHEAD()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("OTHER_MESSENGER()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("PORTAL()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("RELAY_CALL_OVERHEAD()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("RELAY_CONSTANT_OVERHEAD()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("RELAY_GAS_CHECK_BUFFER()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("RELAY_RESERVED_GAS()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("baseGas(bytes,uint32)"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("failedMessages(bytes32)"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("initialize(address)"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("messageNonce()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("paused()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("portal()"), false);
        _addSpec(
            "L1CrossDomainMessenger", _getSel("relayMessage(uint256,address,address,uint256,uint256,bytes)"), false
        );
        _addSpec("L1CrossDomainMessenger", _getSel("sendMessage(address,bytes,uint32)"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("successfulMessages(bytes32)"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("superchainConfig()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("version()"), false);
        _addSpec("L1CrossDomainMessenger", _getSel("xDomainMessageSender()"), false);

        // L1ERC721Bridge
        _addSpec("L1ERC721Bridge", _getSel("MESSENGER()"), false);
        _addSpec("L1ERC721Bridge", _getSel("OTHER_BRIDGE()"), false);
        _addSpec("L1ERC721Bridge", _getSel("bridgeERC721(address,address,uint256,uint32,bytes)"), false);
        _addSpec("L1ERC721Bridge", _getSel("bridgeERC721To(address,address,address,uint256,uint32,bytes)"), false);
        _addSpec("L1ERC721Bridge", _getSel("deposits(address,address,uint256)"), false);
        _addSpec("L1ERC721Bridge", _getSel("finalizeBridgeERC721(address,address,address,address,uint256,bytes)"), true);
        _addSpec("L1ERC721Bridge", _getSel("messenger()"), false);
        _addSpec("L1ERC721Bridge", _getSel("otherBridge()"), false);
        _addSpec("L1ERC721Bridge", _getSel("version()"), false);

        // L1StandardBridge
        _addSpec("L1StandardBridge", _getSel("MESSENGER()"), false);
        _addSpec("L1StandardBridge", _getSel("OTHER_BRIDGE()"), false);
        _addSpec("L1StandardBridge", _getSel("bridgeERC20(address,address,uint256,uint32,bytes)"), false);
        _addSpec("L1StandardBridge", _getSel("bridgeERC20To(address,address,address,uint256,uint32,bytes)"), false);
        _addSpec("L1StandardBridge", _getSel("bridgeETH(uint32,bytes)"), false);
        _addSpec("L1StandardBridge", _getSel("bridgeETHTo(address,uint32,bytes)"), false);
        _addSpec("L1StandardBridge", _getSel("depositERC20(address,address,uint256,uint32,bytes)"), false);
        _addSpec("L1StandardBridge", _getSel("depositERC20To(address,address,address,uint256,uint32,bytes)"), false);
        _addSpec("L1StandardBridge", _getSel("depositETH(uint32,bytes)"), false);
        _addSpec("L1StandardBridge", _getSel("depositETHTo(address,uint32,bytes)"), false);
        _addSpec("L1StandardBridge", _getSel("deposits(address,address)"), false);
        _addSpec(
            "L1StandardBridge", _getSel("finalizeBridgeERC20(address,address,address,address,uint256,bytes)"), true
        );
        _addSpec("L1StandardBridge", _getSel("finalizeBridgeETH(address,address,uint256,bytes)"), true);
        _addSpec(
            "L1StandardBridge", _getSel("finalizeERC20Withdrawal(address,address,address,address,uint256,bytes)"), true
        );
        _addSpec("L1StandardBridge", _getSel("finalizeETHWithdrawal(address,address,uint256,bytes)"), true);
        _addSpec("L1StandardBridge", _getSel("initialize(address)"), false);
        _addSpec("L1StandardBridge", _getSel("l2TokenBridge()"), false);
        _addSpec("L1StandardBridge", _getSel("messenger()"), false);
        _addSpec("L1StandardBridge", _getSel("otherBridge()"), false);
        _addSpec("L1StandardBridge", _getSel("paused()"), false);
        _addSpec("L1StandardBridge", _getSel("superchainConfig()"), false);
        _addSpec("L1StandardBridge", _getSel("version()"), false);

        // L2OutputOracle
        _addSpec("L2OutputOracle", _getSel("CHALLENGER()"), false);
        _addSpec("L2OutputOracle", _getSel("FINALIZATION_PERIOD_SECONDS()"), false);
        _addSpec("L2OutputOracle", _getSel("L2_BLOCK_TIME()"), false);
        _addSpec("L2OutputOracle", _getSel("PROPOSER()"), false);
        _addSpec("L2OutputOracle", _getSel("SUBMISSION_INTERVAL()"), false);
        _addSpec("L2OutputOracle", _getSel("challenger()"), false);
        _addSpec("L2OutputOracle", _getSel("computeL2Timestamp(uint256)"), false);
        _addSpec("L2OutputOracle", _getSel("deleteL2Outputs(uint256)"), true);
        _addSpec("L2OutputOracle", _getSel("finalizationPeriodSeconds()"), false);
        _addSpec("L2OutputOracle", _getSel("getL2Output(uint256)"), false);
        _addSpec("L2OutputOracle", _getSel("getL2OutputAfter(uint256)"), false);
        _addSpec("L2OutputOracle", _getSel("getL2OutputIndexAfter(uint256)"), false);
        _addSpec("L2OutputOracle", _getSel("initialize(uint256,uint256)"), false);
        _addSpec("L2OutputOracle", _getSel("l2BlockTime()"), false);
        _addSpec("L2OutputOracle", _getSel("latestBlockNumber()"), false);
        _addSpec("L2OutputOracle", _getSel("latestOutputIndex()"), false);
        _addSpec("L2OutputOracle", _getSel("nextBlockNumber()"), false);
        _addSpec("L2OutputOracle", _getSel("nextOutputIndex()"), false);
        _addSpec("L2OutputOracle", _getSel("proposeL2Output(bytes32,uint256,bytes32,uint256)"), true);
        _addSpec("L2OutputOracle", _getSel("proposer()"), false);
        _addSpec("L2OutputOracle", _getSel("startingBlockNumber()"), false);
        _addSpec("L2OutputOracle", _getSel("startingTimestamp()"), false);
        _addSpec("L2OutputOracle", _getSel("submissionInterval()"), false);
        _addSpec("L2OutputOracle", _getSel("version()"), false);

        // OptimismPortal
        _addSpec("OptimismPortal", _getSel("GUARDIAN()"), false);
        _addSpec("OptimismPortal", _getSel("L2_ORACLE()"), false);
        _addSpec("OptimismPortal", _getSel("SYSTEM_CONFIG()"), false);
        _addSpec("OptimismPortal", _getSel("depositTransaction(address,uint256,uint64,bool,bytes)"), false);
        _addSpec("OptimismPortal", _getSel("donateETH()"), false);
        _addSpec("OptimismPortal", OptimismPortal.finalizeWithdrawalTransaction.selector, true); // pauseable
        _addSpec("OptimismPortal", _getSel("finalizedWithdrawals(bytes32)"), false);
        _addSpec("OptimismPortal", _getSel("guardian()"), false);
        _addSpec("OptimismPortal", _getSel("initialize(address)"), false);
        _addSpec("OptimismPortal", _getSel("isOutputFinalized(uint256)"), false);
        _addSpec("OptimismPortal", _getSel("l2Oracle()"), false);
        _addSpec("OptimismPortal", _getSel("l2Sender()"), false);
        _addSpec("OptimismPortal", _getSel("minimumGasLimit(uint64)"), false);
        _addSpec("OptimismPortal", _getSel("params()"), false);
        _addSpec("OptimismPortal", _getSel("paused()"), false);
        _addSpec("OptimismPortal", OptimismPortal.proveWithdrawalTransaction.selector, true); // pauseable
        _addSpec("OptimismPortal", _getSel("provenWithdrawals(bytes32)"), false);
        _addSpec("OptimismPortal", _getSel("superchainConfig()"), false);
        _addSpec("OptimismPortal", _getSel("systemConfig()"), false);
        _addSpec("OptimismPortal", _getSel("version()"), false);

        // ProtocolVersions
        _addSpec("ProtocolVersions", _getSel("RECOMMENDED_SLOT()"), false);
        _addSpec("ProtocolVersions", _getSel("REQUIRED_SLOT()"), false);
        _addSpec("ProtocolVersions", _getSel("VERSION()"), false);
        _addSpec("ProtocolVersions", ProtocolVersions.initialize.selector, false);
        _addSpec("ProtocolVersions", _getSel("owner()"), false);
        _addSpec("ProtocolVersions", ProtocolVersions.recommended.selector, false);
        _addSpec("ProtocolVersions", _getSel("renounceOwnership()"), true);
        _addSpec("ProtocolVersions", ProtocolVersions.required.selector, false);
        _addSpec("ProtocolVersions", ProtocolVersions.setRequired.selector, true);
        _addSpec("ProtocolVersions", ProtocolVersions.setRecommended.selector, true);
        _addSpec("ProtocolVersions", _getSel("transferOwnership(address)"), false);
        _addSpec("ProtocolVersions", _getSel("version()"), false);

        // ResourceMetering
        _addSpec("ResourceMetering", _getSel("params()"), false);

        // SuperchainConfig
        _addSpec("SuperchainConfig", _getSel("GUARDIAN_SLOT()"), false);
        _addSpec("SuperchainConfig", _getSel("PAUSED_SLOT()"), false);
        _addSpec("SuperchainConfig", _getSel("guardian()"), false);
        _addSpec("SuperchainConfig", _getSel("initialize(address)"), false);
        _addSpec("SuperchainConfig", _getSel("pause(string)"), true);
        _addSpec("SuperchainConfig", _getSel("paused()"), false);
        _addSpec("SuperchainConfig", _getSel("unpause()"), true);
        _addSpec("SuperchainConfig", _getSel("version()"), false);

        // SystemConfig
        _addSpec("SystemConfig", _getSel("UNSAFE_BLOCK_SIGNER_SLOT()"), false);
        _addSpec("SystemConfig", _getSel("VERSION()"), false);
        _addSpec("SystemConfig", _getSel("batcherHash()"), false);
        _addSpec("SystemConfig", _getSel("gasLimit()"), false);
        _addSpec("SystemConfig", SystemConfig.initialize.selector, false);
        _addSpec("SystemConfig", SystemConfig.minimumGasLimit.selector, false);
        _addSpec("SystemConfig", _getSel("overhead()"), false);
        _addSpec("SystemConfig", _getSel("owner()"), false);
        _addSpec("SystemConfig", _getSel("renounceOwnership()"), true);
        _addSpec("SystemConfig", SystemConfig.resourceConfig.selector, false);
        _addSpec("SystemConfig", _getSel("scalar()"), false);
        _addSpec("SystemConfig", SystemConfig.setBatcherHash.selector, true);
        _addSpec("SystemConfig", SystemConfig.setGasConfig.selector, true);
        _addSpec("SystemConfig", SystemConfig.setGasLimit.selector, true);
        _addSpec("SystemConfig", SystemConfig.setResourceConfig.selector, true);
        _addSpec("SystemConfig", SystemConfig.setUnsafeBlockSigner.selector, true);
        _addSpec("SystemConfig", _getSel("transferOwnership(address)"), true);
        _addSpec("SystemConfig", SystemConfig.unsafeBlockSigner.selector, false);
        _addSpec("SystemConfig", _getSel("version()"), false);
    }

    /// @dev Computes the selector from a function signature.
    function _getSel(string memory _name) internal returns (bytes4) {
        return bytes4(keccak256(abi.encodePacked(_name)));
    }

    /// @dev Adds a spec for a function.
    function _addSpec(string memory _name, bytes4 _sel, bool _auth) internal {
        specs[_name][_sel] = Spec({ name: _name, sel: _sel, auth: _auth });
        numEntries[_name]++;
    }

    /// @notice Ensures that there's an auth spec for every L1 contract function.
    function testContractAuth() public {
        Abi[] memory abis = _getL1ContractFunctionAbis();

        for (uint256 i = 0; i < abis.length; i++) {
            string memory contractName = abis[i].contractName;
            assertEq(
                abis[i].entries.length, numEntries[contractName], "Authorization_Test: invalid number of ABI entries"
            );

            for (uint256 j = 0; j < abis[i].entries.length; j++) {
                AbiEntry memory abiEntry = abis[i].entries[j];
                console.log(
                    "Checking auth spec for %s: %s(%x)", contractName, abiEntry.fnName, uint256(uint32(abiEntry.sel))
                );
                Spec memory spec = specs[contractName][abiEntry.sel];
                assertTrue(spec.sel != bytes4(0), "Authorization_Test: missing spec definition");
                assertEq(abiEntry.sel, spec.sel, "Authorization_Test: invalid ABI");
            }
        }
    }

    /// @dev Returns the function ABIs of all L1 contracts.
    function _getL1ContractFunctionAbis() internal returns (Abi[] memory abis_) {
        string[] memory command = new string[](3);
        command[0] = Executables.bash;
        command[1] = "-c";
        command[2] = string.concat(
            Executables.find,
            " src/L1 -type f -exec basename {} \\;",
            " | ",
            Executables.sed,
            " 's/\\.[^.]*$//'",
            " | ",
            Executables.jq,
            " -R -s 'split(\"\n\")[:-1]'"
        );
        string[] memory contractNames = abi.decode(vm.parseJson(string(vm.ffi(command))), (string[]));

        abis_ = new Abi[](contractNames.length);

        for (uint256 i; i < contractNames.length; i++) {
            string memory contractName = contractNames[i];
            string[] memory methodIdentifiers = deploy.getMethodIdentifiers(contractName);
            abis_[i].contractName = contractName;
            abis_[i].entries = new AbiEntry[](methodIdentifiers.length);
            for (uint256 j; j < methodIdentifiers.length; j++) {
                string memory fnName = methodIdentifiers[j];
                bytes4 sel = bytes4(keccak256(abi.encodePacked(fnName)));
                abis_[i].entries[j] = AbiEntry({ fnName: fnName, sel: sel });
            }
        }
    }
}
