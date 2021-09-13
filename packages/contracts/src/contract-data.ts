/* eslint-disable @typescript-eslint/no-var-requires */
import { predeploys as l2Addresses } from './predeploys'
import { Network } from './connect-contracts'

/**
 * This file is necessarily not DRY because it needs to be usable
 * in a browser context and can't take advantage of dynamic imports
 * (ie: the json needs to all be imported when transpiled)
 */

const Mainnet__Lib_AddressManager = require('../deployments/mainnet/Lib_AddressManager.json')
const Mainnet__OVM_CanonicalTransactionChain = require('../deployments/mainnet/OVM_CanonicalTransactionChain.json')
const Mainnet__OVM_L1CrossDomainMessenger = require('../deployments/mainnet/L1CrossDomainMessenger.json')
const Mainnet__OVM_StateCommitmentChain = require('../deployments/mainnet/OVM_StateCommitmentChain.json')
const Mainnet__Proxy__OVM_L1CrossDomainMessenger = require('../deployments/mainnet/Proxy__OVM_L1CrossDomainMessenger.json')
const Mainnet__OVM_BondManager = require('../deployments/mainnet/mockOVM_BondManager.json')

const Kovan__Lib_AddressManager = require('../deployments/kovan/Lib_AddressManager.json')
const Kovan__OVM_CanonicalTransactionChain = require('../deployments/kovan/OVM_CanonicalTransactionChain.json')
const Kovan__OVM_L1CrossDomainMessenger = require('../deployments/kovan/L1CrossDomainMessenger.json')
const Kovan__OVM_StateCommitmentChain = require('../deployments/kovan/OVM_StateCommitmentChain.json')
const Kovan__Proxy__OVM_L1CrossDomainMessenger = require('../deployments/kovan/Proxy__OVM_L1CrossDomainMessenger.json')
const Kovan__OVM_BondManager = require('../deployments/kovan/mockOVM_BondManager.json')

const Goerli__Lib_AddressManager = require('../deployments/goerli/Lib_AddressManager.json')
const Goerli__OVM_CanonicalTransactionChain = require('../deployments/goerli/OVM_CanonicalTransactionChain.json')
const Goerli__OVM_L1CrossDomainMessenger = require('../deployments/goerli/L1CrossDomainMessenger.json')
const Goerli__OVM_StateCommitmentChain = require('../deployments/goerli/OVM_StateCommitmentChain.json')
const Goerli__Proxy__OVM_L1CrossDomainMessenger = require('../deployments/goerli/Proxy__OVM_L1CrossDomainMessenger.json')
const Goerli__OVM_BondManager = require('../deployments/goerli/mockOVM_BondManager.json')

export const getL1ContractData = (network: Network) => {
  return {
    Lib_AddressManager: {
      mainnet: Mainnet__Lib_AddressManager,
      kovan: Kovan__Lib_AddressManager,
      goerli: Goerli__Lib_AddressManager,
    }[network],
    OVM_CanonicalTransactionChain: {
      mainnet: Mainnet__OVM_CanonicalTransactionChain,
      kovan: Kovan__OVM_CanonicalTransactionChain,
      goerli: Goerli__OVM_CanonicalTransactionChain,
    }[network],
    L1CrossDomainMessenger: {
      mainnet: Mainnet__OVM_L1CrossDomainMessenger,
      kovan: Kovan__OVM_L1CrossDomainMessenger,
      goerli: Goerli__OVM_L1CrossDomainMessenger,
    }[network],
    OVM_StateCommitmentChain: {
      mainnet: Mainnet__OVM_StateCommitmentChain,
      kovan: Kovan__OVM_StateCommitmentChain,
      goerli: Goerli__OVM_StateCommitmentChain,
    }[network],
    Proxy__OVM_L1CrossDomainMessenger: {
      mainnet: Mainnet__Proxy__OVM_L1CrossDomainMessenger,
      kovan: Kovan__Proxy__OVM_L1CrossDomainMessenger,
      goerli: Goerli__Proxy__OVM_L1CrossDomainMessenger,
    }[network],
    OVM_BondManager: {
      mainnet: Mainnet__OVM_BondManager,
      kovan: Kovan__OVM_BondManager,
      goerli: Goerli__OVM_BondManager,
    }[network],
  }
}

const OVM_ETH = require('../artifacts/contracts/L2/predeploys/OVM_ETH.sol/OVM_ETH.json')
const L2CrossDomainMessenger = require('../artifacts/contracts/L2/messaging/L2CrossDomainMessenger.sol/L2CrossDomainMessenger.json')
const OVM_L2ToL1MessagePasser = require('../artifacts/contracts/L2/predeploys/OVM_L2ToL1MessagePasser.sol/OVM_L2ToL1MessagePasser.json')
const OVM_L1MessageSender = require('../artifacts/contracts/L2/predeploys/iOVM_L1MessageSender.sol/iOVM_L1MessageSender.json')
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
    OVM_L1MessageSender: {
      abi: OVM_L1MessageSender.abi,
      address: l2Addresses.OVM_L1MessageSender,
    },
    OVM_DeployerWhitelist: {
      abi: OVM_DeployerWhitelist.abi,
      address: l2Addresses.OVM_DeployerWhitelist,
    },
  }
}
