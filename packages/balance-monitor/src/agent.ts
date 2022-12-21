import {
  BlockEvent,
  Finding,
  HandleBlock,
  FindingSeverity,
  FindingType,
  getEthersProvider,
} from 'forta-agent'
import { utils, BigNumber as BN, providers } from 'ethers'

type AccountAlert = {
  name: string
  address: string
  thresholds: {
    warning: BN
    danger: BN
  }
}

export const accounts: AccountAlert[] = [
  {
    name: 'Sequencer',
    address: process.env.SEQUENCER_ADDRESS,
    thresholds: {
      warning: BN.from(process.env.SEQUENCER_WARNING_THRESHOLD),
      danger: BN.from(process.env.SEQUENCER_DANGER_THRESHOLD),
    },
  },
  {
    name: 'Proposer',
    address: process.env.PROPOSER_ADDRESS,
    thresholds: {
      warning: BN.from(process.env.PROPOSER_WARNING_THRESHOLD),
      danger: BN.from(process.env.PROPOSER_DANGER_THRESHOLD),
    },
  },
]

const ethersProvider = new providers.JsonRpcProvider(
  process.env.L1_RPC_URL
)

function provideHandleBlock(
  ethersProvider: providers.JsonRpcProvider
): HandleBlock {
  return async function handleBlock(blockEvent: BlockEvent) {
    // report finding if specified account balance falls below threshold
    const findings: Finding[] = []

    // iterate over accounts with the index
    for (const [idx, account] of accounts.entries()) {
      const accountBalance = BN.from(
        (
          await ethersProvider.getBalance(
            account.address,
            blockEvent.blockNumber
          )
        ).toString()
      )
      if (accountBalance.gte(account.thresholds.warning)) {
        // return if this is the last account
        if (idx === accounts.length - 1) {
          return findings
        } else {
          continue
        }
      }
    }

    // todo: Add to the findings array when balances are below the threshold

    return findings
  }
}

export default {
  provideHandleBlock,
  handleBlock: provideHandleBlock(ethersProvider),
}
