/* External Imports */
import {
  computeStorageSlots,
  getStorageLayout,
} from '@defi-wonderland/smock/dist/src/utils'
import { remove0x } from '@eth-optimism/core-utils'

/* Internal Imports */
import { predeploys } from './predeploys'
import { getContractArtifact } from './contract-artifacts'

export interface RollupDeployConfig {
  // Address that will own the L2 deployer whitelist.
  whitelistOwner: string
  // Address that will own the L2 gas price oracle.
  gasPriceOracleOwner: string
  // Overhead value of the gas price oracle
  gasPriceOracleOverhead: number
  // Scalar value of the gas price oracle
  gasPriceOracleScalar: number
  // L1 base fee of the gas price oracle
  gasPriceOracleL1BaseFee: number
  // L2 gas price of the gas price oracle
  gasPriceOracleGasPrice: number
  // Number of decimals of the gas price oracle scalar
  gasPriceOracleDecimals: number
  // Initial value for the L2 block gas limit.
  l2BlockGasLimit: number
  // Chain ID to give the L2 network.
  l2ChainId: number
  // Address of the key that will sign blocks.
  blockSignerAddress: string
  // Address of the L1StandardBridge contract.
  l1StandardBridgeAddress: string
  // Address of the L1 fee wallet.
  l1FeeWalletAddress: string
  // Address of the L1CrossDomainMessenger contract.
  l1CrossDomainMessengerAddress: string
}

/**
 * Generates the initial state for the L2 system by injecting the relevant L2 system contracts.
 *
 * @param cfg Configuration for the L2 system.
 * @returns Generated L2 genesis state.
 */
export const makeL2GenesisFile = async (
  cfg: RollupDeployConfig
): Promise<any> => {
  // Very basic validation.
  for (const [key, val] of Object.entries(cfg)) {
    if (val === undefined) {
      throw new Error(`must provide an input for config value: ${key}`)
    }
  }

  const variables = {
    OVM_DeployerWhitelist: {
      owner: cfg.whitelistOwner,
    },
    OVM_GasPriceOracle: {
      _owner: cfg.gasPriceOracleOwner,
      gasPrice: cfg.gasPriceOracleGasPrice,
      l1BaseFee: cfg.gasPriceOracleL1BaseFee,
      overhead: cfg.gasPriceOracleOverhead,
      scalar: cfg.gasPriceOracleScalar,
      decimals: cfg.gasPriceOracleDecimals,
    },
    L2StandardBridge: {
      l1TokenBridge: cfg.l1StandardBridgeAddress,
      messenger: predeploys.L2CrossDomainMessenger,
    },
    OVM_SequencerFeeVault: {
      l1FeeWallet: cfg.l1FeeWalletAddress,
    },
    OVM_ETH: {
      l2Bridge: predeploys.L2StandardBridge,
      _name: 'Ether',
      _symbol: 'ETH',
      _decimals: 18,
    },
    L2CrossDomainMessenger: {
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

  return {
    config: {
      chainId: cfg.l2ChainId,
      homesteadBlock: 0,
      eip150Block: 0,
      eip155Block: 0,
      eip158Block: 0,
      byzantiumBlock: 0,
      constantinopleBlock: 0,
      petersburgBlock: 0,
      istanbulBlock: 0,
      muirGlacierBlock: 0,
      clique: {
        period: 0,
        epoch: 30000,
      },
    },
    difficulty: '1',
    gasLimit: cfg.l2BlockGasLimit.toString(10),
    extradata:
      '0x' +
      '00'.repeat(32) +
      remove0x(cfg.blockSignerAddress) +
      '00'.repeat(65),
    alloc: dump,
  }
}
