import {
  BlockEvent,
  Finding,
  HandleBlock,
  FindingSeverity,
  FindingType,
} from 'forta-agent'
import { BigNumber, providers } from 'ethers'

import { describeFinding } from './utils'

type AccountAlert = {
  name: string
  address: string
  thresholds: {
    warning: BigNumber
    danger: BigNumber
  }
}

export const accounts: AccountAlert[] = [
  {
    name: 'Sequencer',
    address: process.env.SEQUENCER_ADDRESS,
    thresholds: {
      warning: BigNumber.from(process.env.SEQUENCER_WARNING_THRESHOLD),
      danger: BigNumber.from(process.env.SEQUENCER_DANGER_THRESHOLD),
    },
  },
  {
    name: 'Proposer',
    address: process.env.PROPOSER_ADDRESS,
    thresholds: {
      warning: BigNumber.from(process.env.PROPOSER_WARNING_THRESHOLD),
      danger: BigNumber.from(process.env.PROPOSER_DANGER_THRESHOLD),
    },
  },
]

const provideHandleBlock = (
  provider: providers.JsonRpcProvider
): HandleBlock => {
  return async (blockEvent: BlockEvent) => {
    // report finding if specified account balance falls below threshold
    const findings: Finding[] = []

    // iterate over accounts with the index
    for (const [ , account] of accounts.entries()) {
      const accountBalance = BigNumber.from(
        (
          await provider.getBalance(account.address, blockEvent.blockNumber)
        ).toString()
      )
      if (accountBalance.lte(account.thresholds.warning)) {
        findings.push(
          Finding.fromObject({
            name: 'Low Account Balance',
            description: describeFinding(
              account.address,
              accountBalance,
              account.thresholds.warning
            ),
            alertId: `OPTIMISM-BALANCE-WARNING-${account.name}`,
            severity: FindingSeverity.Info,
            type: FindingType.Info,
            metadata: {
              balance: accountBalance.toString(),
            },
          })
        )
      }

      if (accountBalance.lte(account.thresholds.danger)) {
        findings.push(
          Finding.fromObject({
            name: 'Minimum Account Balance',
            description: describeFinding(
              account.address,
              accountBalance,
              account.thresholds.danger
            ),
            alertId: `OPTIMISM-BALANCE-DANGER-${account.name}`,
            severity: FindingSeverity.High,
            type: FindingType.Info,
            metadata: {
              balance: accountBalance.toString(),
            },
          })
        )
      }
    }
    return findings
  }
}

const l1Provider = new providers.JsonRpcProvider(process.env.L1_RPC_URL)

export default {
  provideHandleBlock,
  handleBlock: provideHandleBlock(l1Provider),
}
