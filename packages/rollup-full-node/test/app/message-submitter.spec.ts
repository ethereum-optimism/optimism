/* External Imports */
import { add0x, sleep } from '@eth-optimism/core-utils'
import { L2ToL1Message } from '@eth-optimism/rollup-core'
import * as L2ToL1MessageReceiverContractDefinition from '@eth-optimism/rollup-contracts/artifacts/L2ToL1MessageReceiver.json'

import { Contract, Wallet } from 'ethers'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { DefaultL2ToL1MessageSubmitter } from '../../src/app'
import { Provider } from 'ethers/providers'

describe('L2 to L1 Message Submitter', () => {
  let provider: Provider
  let wallet: Wallet
  let messageReceiver: Contract
  let messageSubmitter: DefaultL2ToL1MessageSubmitter

  beforeEach(async () => {
    provider = createMockProvider()
    wallet = getWallets(provider)[0]
    messageReceiver = await deployContract(
      wallet,
      L2ToL1MessageReceiverContractDefinition,
      [wallet.address, 1]
    )
    messageSubmitter = await DefaultL2ToL1MessageSubmitter.create(
      wallet,
      messageReceiver
    )
  })

  it('Submits messages to L1', async () => {
    const l1ToL2Message: L2ToL1Message = {
      nonce: 0,
      ovmSender: Wallet.createRandom().address,
      callData: add0x(Buffer.from('1.21 Gigawatts!?!').toString('hex')),
    }
    await messageSubmitter.submitMessage(l1ToL2Message)

    messageSubmitter
      .getHighestNonceSubmitted()
      .should.equal(0, 'Message not submitted!')

    await sleep(1000)

    messageSubmitter
      .getHighestNonceConfirmed()
      .should.equal(0, 'Message not confirmed!')
  })
})
