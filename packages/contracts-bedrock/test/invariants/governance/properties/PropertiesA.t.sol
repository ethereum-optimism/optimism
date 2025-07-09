// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { HandlersParent } from "../handlers/HandlersParent.t.sol";

contract PropertiesA is HandlersParent {
    function property_sanity() public {
        assert(address(validator.GOVERNOR()) == address(governor));
        assert(address(validator.proposalTypesConfigurator()) == address(proposalTypesConfigurator));
        assert(validator.proposalDistributionThreshold() == DISTRIBUTION_THRESHOLD);
        assert(validator.APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID() == APPROVED_PROPOSER_ATTESTATION_SCHEMA_UID);
        assert(validator.TOP_DELEGATES_ATTESTATION_SCHEMA_UID() == TOP_DELEGATES_ATTESTATION_SCHEMA_UID);
    }
}
