// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

// Interfaces
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";

/// @title ComplianceModule
/// @notice Test stub with only the surface DeployCompliance deploys and checks. It carries the
///         real contract's name because the script loads it by artifact name.
contract ComplianceModule is Initializable {
    IOptimismPortal2 public immutable portal;
    IL1StandardBridge public immutable bridge;
    IL1CrossDomainMessenger public immutable l1CrossDomainMessenger;
    address public owner;
    IPolicy public policy;

    constructor(IOptimismPortal2 _portal, IL1StandardBridge _bridge, IL1CrossDomainMessenger _l1CrossDomainMessenger) {
        portal = _portal;
        bridge = _bridge;
        l1CrossDomainMessenger = _l1CrossDomainMessenger;
        _disableInitializers();
    }

    function initialize(address _owner, IPolicy _policy) external initializer {
        owner = _owner;
        policy = _policy;
    }
}

/// @title VisionPolicyV1
/// @notice Test stub with only the surface DeployCompliance deploys and checks. It carries the
///         real contract's name because the script loads it by artifact name.
contract VisionPolicyV1 {
    IComplianceModule public immutable module;

    constructor(IComplianceModule _module) {
        module = _module;
    }
}
