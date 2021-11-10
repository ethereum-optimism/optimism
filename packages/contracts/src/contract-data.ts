/* eslint-disable @typescript-eslint/no-var-requires */
import { predeploys as l2Addresses } from './predeploys'
import { Network } from './connect-contracts'

/**
 * This file is necessarily not DRY because it needs to be usable
 * in a browser context and can't take advantage of dynamic imports
 * (ie: the json needs to all be imported when transpiled)
 */

const Mainnet__Lib_AddressManager = require('../deployments/mainnet/Lib_AddressManager.json')
const Mainnet__CanonicalTransactionChain = require('../deployments/mainnet/CanonicalTransactionChain.json')
const Mainnet__L1CrossDomainMessenger = require('../deployments/mainnet/L1CrossDomainMessenger.json')
const Mainnet__StateCommitmentChain = require('../deployments/mainnet/StateCommitmentChain.json')
const Mainnet__Proxy__L1CrossDomainMessenger = require('../deployments/mainnet/Proxy__L1CrossDomainMessenger.json')
const Mainnet__BondManager = require('../deployments/mainnet/mockBondManager.json')

const Kovan__Lib_AddressManager = require('../deployments/kovan/Lib_AddressManager.json')
const Kovan__CanonicalTransactionChain = require('../deployments/kovan/CanonicalTransactionChain.json')
const Kovan__L1CrossDomainMessenger = require('../deployments/kovan/L1CrossDomainMessenger.json')
const Kovan__StateCommitmentChain = require('../deployments/kovan/StateCommitmentChain.json')
const Kovan__Proxy__L1CrossDomainMessenger = require('../deployments/kovan/Proxy__L1CrossDomainMessenger.json')
const Kovan__BondManager = require('../deployments/kovan/mockBondManager.json')

const Goerli__Lib_AddressManager = require('../deployments/goerli/Lib_AddressManager.json')
const Goerli__CanonicalTransactionChain = require('../deployments/goerli/CanonicalTransactionChain.json')
const Goerli__L1CrossDomainMessenger = require('../deployments/goerli/L1CrossDomainMessenger.json')
const Goerli__StateCommitmentChain = require('../deployments/goerli/StateCommitmentChain.json')
const Goerli__Proxy__L1CrossDomainMessenger = require('../deployments/goerli/Proxy__L1CrossDomainMessenger.json')
const Goerli__BondManager = require('../deployments/goerli/mockBondManager.json')

export const getL1ContractData = (network: Network) => {
  return {
    Lib_AddressManager: {
      mainnet: Mainnet__Lib_AddressManager,
      kovan: Kovan__Lib_AddressManager,
      goerli: Goerli__Lib_AddressManager,
    }[network],
    CanonicalTransactionChain: {
      mainnet: Mainnet__CanonicalTransactionChain,
      kovan: Kovan__CanonicalTransactionChain,
      goerli: Goerli__CanonicalTransactionChain,
    }[network],
    L1CrossDomainMessenger: {
      mainnet: Mainnet__L1CrossDomainMessenger,
      kovan: Kovan__L1CrossDomainMessenger,
      goerli: Goerli__L1CrossDomainMessenger,
    }[network],
    StateCommitmentChain: {
      mainnet: Mainnet__StateCommitmentChain,
      kovan: Kovan__StateCommitmentChain,
      goerli: Goerli__StateCommitmentChain,
    }[network],
    Proxy__L1CrossDomainMessenger: {
      mainnet: Mainnet__Proxy__L1CrossDomainMessenger,
      kovan: Kovan__Proxy__L1CrossDomainMessenger,
      goerli: Goerli__Proxy__L1CrossDomainMessenger,
    }[network],
    BondManager: {
      mainnet: Mainnet__BondManager,
      kovan: Kovan__BondManager,
      goerli: Goerli__BondManager,
    }[network],
  }
}

const OVM_ETH = require('../artifacts/contracts/L2/predeploys/OVM_ETH.sol/OVM_ETH.json')
const L2CrossDomainMessenger = require('../artifacts/contracts/L2/messaging/L2CrossDomainMessenger.sol/L2CrossDomainMessenger.json')
const OVM_L2ToL1MessagePasser = require('../artifacts/contracts/L2/predeploys/OVM_L2ToL1MessagePasser.sol/OVM_L2ToL1MessagePasser.json')
const OVM_DeployerWhitelist = require('../artifacts/contracts/L2/predeploys/OVM_DeployerWhitelist.sol/OVM_DeployerWhitelist.json')

export const getL2ContractData = () => {
  return {
    OVM_ETH: {
      abi: OVM_ETH.abi,
      address: l2Addresses.OVM_ETH,
    },
    L2CrossDomainMessenger: {
      abi: L2CrossDomainMessenger.abi,
      address: l2Addresses.L2CrossDomainMessenger,
    },
    OVM_L2ToL1MessagePasser: {
      abi: OVM_L2ToL1MessagePasser.abi,
      address: l2Addresses.OVM_L2ToL1MessagePasser,
    },
    OVM_DeployerWhitelist: {
      abi: OVM_DeployerWhitelist.abi,
      address: l2Addresses.OVM_DeployerWhitelist,
    },
  }
}
