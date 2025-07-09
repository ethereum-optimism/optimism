// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ProposalValidator } from "src/governance/ProposalValidator.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IOptimismGovernor } from "interfaces/governance/IOptimismGovernor.sol";
import { IProposalTypesConfigurator } from "interfaces/governance/IProposalTypesConfigurator.sol";
import { Proxy } from "src/universal/Proxy.sol";

import { MockGovernor } from "./utils/MockGovernor.sol";

import { Test } from "forge-std/Test.sol";

contract Setup is Test {
    address public owner;

    IOptimismGovernor public governor;
    IProposalTypesConfigurator public proposalTypesConfigurator;
    bytes32 public APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID;
    bytes32 public TOP_DELEGATES_ATTESTATION_SCHEMA_UID;

    uint256 public CYCLE_NUMBER;
    uint256 public START_TIMESTAMP;
    uint256 public DURATION;
    uint256 public DISTRIBUTION_LIMIT;
    uint256 public DISTRIBUTION_THRESHOLD;

    ProposalValidator.ProposalType[] public proposalTypes;

    ProposalValidator public validator;
    ProposalValidator public impl;

    function setUp() public {
        owner = makeAddr("owner");
        CYCLE_NUMBER = 1;
        START_TIMESTAMP = 1000000;
        DURATION = 1 days;
        DISTRIBUTION_LIMIT = 20000 ether;
        DISTRIBUTION_THRESHOLD = 10000 ether;

        APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID =
            bytes32(0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef);
        TOP_DELEGATES_ATTESTATION_SCHEMA_UID =
            bytes32(0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890);

        proposalTypes = [
            ProposalValidator.ProposalType.ProtocolOrGovernorUpgrade,
            ProposalValidator.ProposalType.MaintenanceUpgrade,
            ProposalValidator.ProposalType.CouncilMemberElections,
            ProposalValidator.ProposalType.GovernanceFund,
            ProposalValidator.ProposalType.CouncilBudget,
            ProposalValidator.ProposalType.MaintenanceUpgrade,
            ProposalValidator.ProposalType.CouncilMemberElections,
            ProposalValidator.ProposalType.GovernanceFund,
            ProposalValidator.ProposalType.CouncilBudget
        ];

        ProposalValidator.ProposalTypeData[] memory proposalTypesData =
            new ProposalValidator.ProposalTypeData[](proposalTypes.length);
        for (uint256 i = 0; i < proposalTypes.length; i++) {
            proposalTypesData[i] = ProposalValidator.ProposalTypeData({ requiredApprovals: 1, proposalVotingModule: 1 });
        }

        governor = IOptimismGovernor(address(new MockGovernor()));

        validator = ProposalValidator(address(new Proxy(owner)));

        impl = new ProposalValidator(
            APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID, TOP_DELEGATES_ATTESTATION_SCHEMA_UID, governor
        );

        vm.prank(owner);
        IProxy(payable(address(validator))).upgradeToAndCall(
            address(impl),
            abi.encodeCall(
                impl.initialize,
                (
                    owner,
                    proposalTypesConfigurator,
                    CYCLE_NUMBER,
                    START_TIMESTAMP,
                    DURATION,
                    DISTRIBUTION_LIMIT,
                    DISTRIBUTION_THRESHOLD,
                    proposalTypes,
                    proposalTypesData
                )
            )
        );
    }
}
