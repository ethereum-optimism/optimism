import type { PonderInteropClient } from '@eth-optimism/ponder-interop/client'
import type { Hono } from 'hono'
import type { Logger } from 'pino'
import { isAddress } from 'viem'

import type { RelayFundsDeposited } from '@/deposit/relayFundsDeposited.js'

export interface BudgetRouteDeps {
  relayFunds: RelayFundsDeposited
  ponderClient: PonderInteropClient
  logger: Logger
}

/**
 * Attaches `GET /budget/:address` to the supplied admin Hono instance.
 *
 * Returns the depositor's cumulative on-chain deposit (cross-chain SUM
 * from ponder) alongside the relayer's private `consumed_wei` ledger, so
 * a UI can render a true remaining-budget figure. All wei values are
 * decimal strings to preserve precision on the wire.
 */
export function attachBudgetRoute(api: Hono, deps: BudgetRouteDeps): void {
  const { relayFunds, ponderClient, logger } = deps

  api.get('/budget/:address', async (c) => {
    const raw = c.req.param('address')
    if (!isAddress(raw)) {
      return c.json({ error: 'invalid address' }, 400)
    }
    const address = raw.toLowerCase()

    let totalDepositedWei = 0n
    try {
      const balance = await ponderClient.getDepositBalance(address)
      totalDepositedWei = BigInt(balance.totalBalance)
    } catch (error) {
      logger.warn(
        { err: error, address },
        'budget route: ponder getDepositBalance failed',
      )
      return c.json({ error: 'failed to fetch on-chain balance' }, 502)
    }

    const consumedWei = relayFunds.getConsumed(address)
    const remainingWei =
      totalDepositedWei > consumedWei ? totalDepositedWei - consumedWei : 0n

    return c.json({
      depositor: address,
      totalDepositedWei: totalDepositedWei.toString(),
      consumedWei: consumedWei.toString(),
      remainingWei: remainingWei.toString(),
    })
  })
}
