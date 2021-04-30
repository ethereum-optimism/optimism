import { Signer, Contract, providers } from 'ethers'
import {
  getL1ContractData,
  Layer1ContractsType,
} from '../deployments/contract-data'

const l2Addresses = {
  OVM_ETH: '0x4200000000000000000000000000000000000006',
  OVM_L2CrossDomainMessenger: '0x4200000000000000000000000000000000000007',
  OVM_L2ToL1MessagePasser: '0x4200000000000000000000000000000000000000',
  OVM_L1MessageSender: '0x4200000000000000000000000000000000000001',
  OVM_DeployerWhitelist: '0x4200000000000000000000000000000000000002',
  OVM_ECDSAContractAccount: '0x4200000000000000000000000000000000000003',
  OVM_SequencerEntrypoint: '0x4200000000000000000000000000000000000005',
  Lib_AddressManager: '0x4200000000000000000000000000000000000008',
  ERC1820Registry: '0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24',
}

// // L2 Contract ABIs
// import OVM_ETH from '../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_ETH.sol/OVM_ETH.json'
// import OVM_L2CrossDomainMessenger from '../artifacts-ovm/contracts/optimistic-ethereum/OVM/bridge/messaging/OVM_L2CrossDomainMessenger.sol/OVM_L2CrossDomainMessenger.json'
// import OVM_L1MessageSender from '../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_L1MessageSender.sol/OVM_L1MessageSender.json'
// import OVM_L2ToL1MessagePasser from '../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_L2ToL1MessagePasser.sol/OVM_L2ToL1MessagePasser.json'
// import OVM_DeployerWhitelist from '../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_DeployerWhitelist.sol/OVM_DeployerWhitelist.json'
// import OVM_SequencerEntrypoint from '../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/OVM_SequencerEntrypoint.sol/OVM_SequencerEntrypoint.json'
// import OVM_ECDSAContractAccount from '../artifacts-ovm/contracts/optimistic-ethereum/OVM/accounts/OVM_ECDSAContractAccount.sol/OVM_ECDSAContractAccount.json'
// import Lib_L2AddressManager from '../artifacts-ovm/contracts/optimistic-ethereum/libraries/resolver/Lib_AddressManager.sol/Lib_AddressManager.json'
// import ERC1820Registry from '../artifacts-ovm/contracts/optimistic-ethereum/OVM/predeploys/ERC1820Registry.sol/ERC1820Registry.json'

// export interface Layer2ContractsType {
//   ovmETH: typeof OVM_ETH.abi
//   xDomainMessenger: typeof OVM_L2CrossDomainMessenger.abi
//   l1MessageSender: typeof OVM_L1MessageSender.abi
//   l2MessagePasser: typeof OVM_L2ToL1MessagePasser.abi
//   deployerWhiteList: typeof OVM_DeployerWhitelist.abi
//   sequencerEntryPoint: typeof OVM_SequencerEntrypoint.abi
//   ecdsaContract: typeof OVM_ECDSAContractAccount.abi
//   l2AddressManager: typeof Lib_L2AddressManager.abi
//   erc1820Registry: typeof ERC1820Registry.abi
// }

export const connectContracts = async (
  signerOrProvider: Signer | providers.Provider,
  network: 'goerli' | 'kovan' | 'mainnet'
): Promise<Object> => {
  const contractData = getL1ContractData(network)

  return {
    addressManager: new Contract(
      contractData.Lib_L1AddressManager.abi,
      contractData.Lib_L1AddressManager.address,
      signerOrProvider
    ),
  }
}
