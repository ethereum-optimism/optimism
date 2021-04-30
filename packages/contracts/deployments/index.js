"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getL1ContractData = void 0;
const getL1ContractData = (network) => {
    return {
        Lib_L1AddressManager: require(`../deployments/${network}-v2/Lib_AddressManager.json`),
        OVM_CanonicalTransactionChain: require(`../deployments/${network}-v2/OVM_CanonicalTransactionChain.json`),
        OVM_ExecutionManager: require(`../deployments/${network}-v2/OVM_ExecutionManager.json`),
        OVM_FraudVerifier: require(`../deployments/${network}-v2/OVM_FraudVerifier.json`),
        OVM_L1CrossDomainMessenger: require(`../deployments/${network}-v2/OVM_L1CrossDomainMessenger.json`),
        OVM_L1ETHGateway: require(`../deployments/${network}-v2/OVM_L1ETHGateway.json`),
        OVM_L1MultiMessageRelayer: require(`../deployments/${network}-v2/OVM_L1MultiMessageRelayer.json`),
        OVM_SafetyChecker: require(`../deployments/${network}-v2/OVM_SafetyChecker.json`),
        OVM_StateCommitmentChain: require(`../deployments/${network}-v2/OVM_StateCommitmentChain.json`),
        OVM_StateManagerFactory: require(`../deployments/${network}-v2/OVM_StateManagerFactory.json`),
        OVM_StateTransitionerFactory: require(`../deployments/${network}-v2/OVM_StateTransitionerFactory.json`),
        Proxy__OVM_L1CrossDomainMessenger: require(`../deployments/${network}-v2/Proxy__OVM_L1CrossDomainMessenger.json`),
        Proxy__OVM_L1ETHGateway: require(`../deployments/${network}-v2/Proxy__OVM_L1ETHGateway.json`),
        mockOVM_BondManager: require(`../deployments/${network}-v2/mockOVM_BondManager.json`),
    };
};
exports.getL1ContractData = getL1ContractData;
//# sourceMappingURL=index.js.map