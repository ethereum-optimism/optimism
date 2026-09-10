import { gasTankAbi } from '@eth-optimism/viem/abis/experimental'
import {
  relayCrossDomainMessage,
  simulateRelayCrossDomainMessage,
} from '@eth-optimism/viem/actions/interop'
import type { Logger } from 'pino'
import type { Address, Client, Hex, PublicClient, WalletClient } from 'viem'
import { simulateContract, writeContract } from 'viem/actions'

import type { RevertClassification } from '@/errors.js'
import {
  classifyGasTankRevert,
  classifyL2ToL2CdmRevert,
  RelayError,
} from '@/errors.js'
import type { RelayMessageParams } from '@/strategies/types.js'

/**
 * On-chain submission for a single relay message: the simulate → broadcast
 * pipeline for both destinations (L2ToL2CrossDomainMessenger directly, or
 * the GasTank wrapper when configured). Extracted from Relayer so the cycle
 * orchestration (fetch, GC, module loop) and the tx mechanics live apart —
 * the Relayer only decides WHICH path to take (see submitRelayMessage).
 *
 * Every simulate/broadcast call runs through runStage(), which converts any
 * thrown error into a typed RelayError carrying {stage, reason} for the
 * metrics/registry pipeline.
 */

export async function relayViaL2ToL2CrossDomainMessenger({
  publicClient,
  walletClient,
  params,
  msgLog,
}: {
  publicClient: PublicClient
  walletClient: Client
  params: RelayMessageParams
  msgLog: Logger
}): Promise<Hex> {
  msgLog.info({ messageId: params.id }, 'L2ToL2CDM: simulating')
  const sim = await runStage(
    () => simulateRelayCrossDomainMessage(publicClient, params),
    'simulation',
    classifyL2ToL2CdmRevert,
    msgLog,
    'L2ToL2CDM: simulation failed',
  )

  // submit (skip local gas estimation if sponsored or caller supplied gas)
  let gas: bigint | null | undefined
  if (params.estimatedGasCost !== undefined) gas = params.estimatedGasCost
  // gas is estimated ahead of time (e.g. EthBridgeModule)
  else if (!walletClient.account) gas = null // gas will be sponsored
  else {
    // eth_estimateGas finds the minimum gas at which the OUTER tx succeeds.
    // Promise callbacks execute inside try/catch, so "success" includes the
    // callback itself starving at the 63/64 boundary (it then rejects with
    // CallbackExecuted(false, "") and no error data). Pad the simulated gas
    // so relayed callbacks get real headroom instead of the bare minimum.
    const simGas = (sim as { request?: { gas?: bigint } } | undefined)?.request
      ?.gas
    gas = simGas !== undefined ? (simGas * 3n) / 2n : undefined
  }

  msgLog.info(
    { hasAccount: !!walletClient.account, gas },
    'L2ToL2CDM: submitting relay tx',
  )
  const txHash = await runStage(
    () => relayCrossDomainMessage(walletClient, { ...params, gas }),
    'broadcast',
    classifyL2ToL2CdmRevert,
    msgLog,
    'L2ToL2CDM: relay tx failed',
  )
  msgLog.info({ tx_hash: txHash }, 'L2ToL2CDM: relay tx submitted')
  return txHash
}

export async function relayViaGasTank({
  publicClient,
  walletClient,
  gasTankAddress,
  params,
  msgLog,
}: {
  publicClient: PublicClient
  walletClient: WalletClient
  gasTankAddress: Address
  params: RelayMessageParams
  msgLog: Logger
}): Promise<Hex> {
  const { id, payload, account, accessList, chain } = params
  const relayCall = {
    abi: gasTankAbi,
    address: gasTankAddress,
    functionName: 'relayMessage' as const,
    args: [id, payload] as const,
    account,
    accessList,
  }

  await runStage(
    () => simulateContract(publicClient, relayCall),
    'simulation',
    classifyGasTankRevert,
    msgLog,
    'GasTank: simulation failed',
  )
  return runStage(
    () => writeContract(walletClient, { ...relayCall, chain }),
    'broadcast',
    classifyGasTankRevert,
    msgLog,
    'GasTank: relay tx failed',
  )
}

/**
 * Run a relay sub-step and convert any thrown error into a typed RelayError.
 * Wraps the four simulate/broadcast call sites (L2ToL2CDM × GasTank) with one
 * shape for classification, structured logging, and error promotion.
 */
async function runStage<T>(
  fn: () => Promise<T>,
  stage: 'simulation' | 'broadcast',
  classify: (err: unknown) => RevertClassification | undefined,
  msgLog: Logger,
  failMessage: string,
): Promise<T> {
  try {
    return await fn()
  } catch (error) {
    const classification = classify(error)
    const reason = classification?.reason ?? 'unknown'
    msgLog.warn(
      { err: error, stage, reason, errorName: classification?.errorName },
      failMessage,
    )
    throw new RelayError({
      stage,
      reason,
      cause: error instanceof Error ? error : undefined,
    })
  }
}
