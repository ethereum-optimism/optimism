import { contracts as addrs } from '@eth-optimism/viem'
import { l2ToL2CrossDomainMessengerAbi } from '@eth-optimism/viem/abis'
import { gasTankAbi } from '@eth-optimism/viem/abis/experimental'
import { createConfig, mergeAbis } from 'ponder'
import type { Transport } from 'viem'

import { callbackAbi } from '@/abis/callbackAbi'
import { l2ToL2CrossDomainMessengerAbi as l2ToL2CrossDomainMessengerDevnetAbi } from '@/abis/devnetAbis'
import { promiseAbi } from '@/abis/promiseAbi'
import { relayDepositAbi } from '@/abis/relayDepositAbi'
import { envVars } from '@/constants/envVars.js'

export type Endpoint = {
  id: number
  rpc: Transport
  disableCache?: boolean
}

/**
 * A map of name to {@link Endpoint}
 */
export type Endpoints = Record<string, Endpoint>

/**
 * Log contract configuration on startup
 */
function logContractConfiguration() {
  console.log('\n📋 Contract Configuration:')

  // L2ToL2CrossDomainMessenger (always the same)
  console.log(
    `📮 L2ToL2CrossDomainMessenger: ${addrs.l2ToL2CrossDomainMessenger.address}`,
  )

  // Gas Tank contract
  if (envVars.GAS_TANK_CONTRACT_ADDRESS) {
    console.log(`⛽ Gas Tank Contract: ${envVars.GAS_TANK_CONTRACT_ADDRESS}`)
  } else {
    console.log(
      `⛽ Gas Tank Contract: ❌ NOT CONFIGURED (set GAS_TANK_CONTRACT_ADDRESS)`,
    )
  }

  // Deposit contract
  if (envVars.DEPOSIT_CONTRACT_ADDRESS) {
    console.log(`💰 Deposit Contract: ${envVars.DEPOSIT_CONTRACT_ADDRESS}`)
  } else {
    console.log(
      `💰 Deposit Contract: ❌ NOT CONFIGURED (set DEPOSIT_CONTRACT_ADDRESS)`,
    )
  }

  // Promise contract
  if (envVars.PROMISE_CONTRACT_ADDRESS) {
    console.log(`🤝 Promise Contract: ${envVars.PROMISE_CONTRACT_ADDRESS}`)
  } else {
    console.log(
      `🤝 Promise Contract: ❌ NOT CONFIGURED (set PROMISE_CONTRACT_ADDRESS)`,
    )
  }

  // Callback contract
  if (envVars.CALLBACK_CONTRACT_ADDRESS) {
    console.log(`📞 Callback Contract: ${envVars.CALLBACK_CONTRACT_ADDRESS}`)
  } else {
    console.log(
      `📞 Callback Contract: ❌ NOT CONFIGURED (set CALLBACK_CONTRACT_ADDRESS)`,
    )
  }

  console.log('')
}

/**
 * Create a Ponder config for interop
 * @param endpoints - Interoperable endpoints to index -- {@link Endpoints}
 * @returns Ponder configuration
 */
export function createPonderConfig(endpoints: Endpoints) {
  if (Object.keys(endpoints).length === 0) {
    throw new Error('no endpoints provided')
  }

  // Long-lived dev chains (supersim) prune historical state, so a re-index
  // from block 1 dies on pinned eth_calls. Set PONDER_START_BLOCK to a recent
  // block to restart against a chain that has outlived the node's history.
  const startBlock = Number(process.env.PONDER_START_BLOCK ?? 1)

  // Log contract configuration on startup
  logContractConfiguration()

  // relevant interop contracts
  const contracts = {
    L2ToL2CDM: {
      abi: mergeAbis([
        l2ToL2CrossDomainMessengerAbi,
        l2ToL2CrossDomainMessengerDevnetAbi,
      ]),
      startBlock,
      chain: Object.fromEntries(
        Object.keys(endpoints).map((key) => [
          key,
          { address: addrs.l2ToL2CrossDomainMessenger.address },
        ]),
      ),
    },
    GasTank: {
      abi: gasTankAbi,
      startBlock,
      chain: Object.fromEntries(
        Object.keys(endpoints).map((key) => [
          key,
          { address: envVars.GAS_TANK_CONTRACT_ADDRESS },
        ]),
      ),
    },
    MostBasicRelayDeposit: {
      abi: relayDepositAbi,
      startBlock,
      chain: Object.fromEntries(
        Object.keys(endpoints).map((key) => [
          key,
          { address: envVars.DEPOSIT_CONTRACT_ADDRESS },
        ]),
      ),
    },
    // Promise + Callback are registered on *every* chain in the mesh: the
    // relayer's unshared-resolved computation depends on a resolved promise row
    // appearing on the destination chain (receiveSharedPromise → PromiseResolved).
    Promise: {
      abi: promiseAbi,
      startBlock,
      chain: Object.fromEntries(
        Object.keys(endpoints).map((key) => [
          key,
          { address: envVars.PROMISE_CONTRACT_ADDRESS },
        ]),
      ),
    },
    Callback: {
      abi: callbackAbi,
      startBlock,
      chain: Object.fromEntries(
        Object.keys(endpoints).map((key) => [
          key,
          { address: envVars.CALLBACK_CONTRACT_ADDRESS },
        ]),
      ),
    },
  }

  return createConfig({
    ordering: 'multichain',
    chains: endpoints,
    contracts,
  })
}
