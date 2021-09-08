/* External Imports */
import { Signer } from 'ethers'
import {
  computeStorageSlots,
  getStorageLayout,
} from '@defi-wonderland/smock/dist/src/utils'

/* Internal Imports */
import { predeploys } from './predeploys'
import { getContractArtifact } from './contract-artifacts'

export interface RollupDeployConfig {
  whitelistConfig: {
    owner: string | Signer
    allowArbitraryContractDeployment: boolean
  }
  gasPriceOracleConfig: {
    owner: string | Signer
    initialGasPrice: number
  }
  l1StandardBridgeAddress: string
  l1FeeWalletAddress: string
  l1CrossDomainMessengerAddress: string
}

/**
 * Generates the initial state for the L2 system by injecting the relevant L2 system contracts.
 *
 * @param cfg Configuration for the L2 system.
 * @returns Generated L2 genesis state.
 */
export const makeStateDump = async (cfg: RollupDeployConfig): Promise<any> => {
  const variables = {
    OVM_DeployerWhitelist: {
      initialized: true,
      allowArbitraryDeployment:
        cfg.whitelistConfig.allowArbitraryContractDeployment,
      owner: cfg.whitelistConfig.owner,
    },
    OVM_GasPriceOracle: {
      _owner: cfg.gasPriceOracleConfig.owner,
      gasPrice: cfg.gasPriceOracleConfig.initialGasPrice,
    },
    OVM_L2StandardBridge: {
      l1TokenBridge: cfg.l1StandardBridgeAddress,
      messenger: predeploys.OVM_L2CrossDomainMessenger,
    },
    OVM_SequencerFeeVault: {
      l1FeeWallet: cfg.l1FeeWalletAddress,
    },
    OVM_ETH: {
      l2Bridge: predeploys.OVM_L2StandardBridge,
      _name: 'Ether',
      _symbol: 'ETH',
      _decimals: 18,
    },
    OVM_L2CrossDomainMessenger: {
      _status: 1,
      l1CrossDomainMessenger: cfg.l1CrossDomainMessengerAddress,
    },
  }

  const dump = {}
  for (const predeployName of Object.keys(predeploys)) {
    const predeployAddress = predeploys[predeployName]
    dump[predeployAddress] = {
      balance: '00',
      storage: {},
    }

    if (predeployName === 'OVM_L1MessageSender') {
      // OVM_L1MessageSender is a special case where we just inject a specific bytecode string.
      // We do this because it uses the custom L1MESSAGESENDER opcode (0x4A) which cannot be
      // directly used in Solidity (yet). This bytecode string simply executes the 0x4A opcode
      // and returns the address given by that opcode.
      dump[predeployAddress].code = '0x4A60005260206000F3'
    } else if (predeployName === 'OVM_L1BlockNumber') {
      // Same as above but for OVM_L1BlockNumber (0x4B).
      dump[predeployAddress].code = '0x4B60005260206000F3'
    } else {
      const artifact = getContractArtifact(predeployName)
      dump[predeployAddress].code = artifact.deployedBytecode
    }

    // Compute and set the required storage slots for each contract that needs it.
    if (predeployName in variables) {
      const storageLayout = await getStorageLayout(predeployName)
      const slots = computeStorageSlots(storageLayout, variables[predeployName])
      for (const slot of slots) {
        dump[predeployAddress].storage[slot.key] = slot.val
      }
    }
  }

  return dump
}
