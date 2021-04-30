import * as path from 'path'

export const getL1ContractData = (network: 'goerli' | 'kovan' | 'mainnet') => {
  return {
    Lib_AddressManager: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/Lib_AddressManager.json`
    )),
    OVM_CanonicalTransactionChain: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_CanonicalTransactionChain.json`
    )),
    OVM_ExecutionManager: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_ExecutionManager.json`
    )),
    OVM_FraudVerifier: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_FraudVerifier.json`
    )),
    OVM_L1CrossDomainMessenger: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_L1CrossDomainMessenger.json`
    )),
    OVM_L1ETHGateway: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_L1ETHGateway.json`
    )),
    OVM_L1MultiMessageRelayer: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_L1MultiMessageRelayer.json`
    )),
    OVM_SafetyChecker: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_SafetyChecker.json`
    )),
    OVM_StateCommitmentChain: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_StateCommitmentChain.json`
    )),
    OVM_StateManagerFactory: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_StateManagerFactory.json`
    )),
    OVM_StateTransitionerFactory: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/OVM_StateTransitionerFactory.json`
    )),
    Proxy__OVM_L1CrossDomainMessenger: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/Proxy__OVM_L1CrossDomainMessenger.json`
    )),
    Proxy__OVM_L1ETHGateway: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/Proxy__OVM_L1ETHGateway.json`
    )),
    mockOVM_BondManager: require(path.resolve(
      __dirname,
      `../deployments/${network}-v2/mockOVM_BondManager.json`
    )),
  }
}

export const getL2ContractData = () => {
  return {
    OVM_ETH: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_ETH.sol/OVM_ETH.json`
      )),
      address: '0x4200000000000000000000000000000000000006',
    },
    OVM_L2CrossDomainMessenger: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/bridge/messaging/OVM_L2CrossDomainMessenger.sol/OVM_L2CrossDomainMessenger.json`
      )),
      address: '0x4200000000000000000000000000000000000007',
    },
    OVM_L2ToL1MessagePasser: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_L2ToL1MessagePasser.sol/OVM_L2ToL1MessagePasser.json`
      )),
      address: '0x4200000000000000000000000000000000000000',
    },
    OVM_L1MessageSender: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_L1MessageSender.sol/OVM_L1MessageSender.json`
      )),
      address: '0x4200000000000000000000000000000000000001',
    },
    OVM_DeployerWhitelist: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_DeployerWhitelist.sol/OVM_DeployerWhitelist.json`
      )),
      address: '0x4200000000000000000000000000000000000002',
    },
    OVM_ECDSAContractAccount: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/accounts/OVM_ECDSAContractAccount.sol/OVM_ECDSAContractAccount.json`
      )),
      address: '0x4200000000000000000000000000000000000003',
    },
    OVM_SequencerEntrypoint: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_SequencerEntrypoint.sol/OVM_SequencerEntrypoint.json`
      )),
      address: '0x4200000000000000000000000000000000000005',
    },
    ERC1820Registry: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/ERC1820Registry.sol/ERC1820Registry.json`
      )),
      address: '0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24',
    },
    Lib_AddressManager: {
      abi: require(path.resolve(
        __dirname,
        `../artifacts-ovm/contracts/optimistic-ethereum/libraries/resolver/Lib_AddressManager.sol/Lib_AddressManager.json`
      )),
      address: '0x4200000000000000000000000000000000000008',
    },
  }
}
