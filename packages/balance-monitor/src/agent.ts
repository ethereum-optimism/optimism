import { BlockEvent, Finding, HandleBlock } from 'forta-agent'
import { BigNumber, providers } from 'ethers'

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
    for (const [idx, account] of accounts.entries()) {
      const accountBalance = BigNumber.from(
        (
          await provider.getBalance(account.address, blockEvent.blockNumber)
        ).toString()
      )
      if (accountBalance.gte(account.thresholds.warning)) {
        // todo: add to the findings array when balances are below the threshold
        // return if this is the last account
        if (idx === accounts.length - 1) {
          return findings
        }
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
