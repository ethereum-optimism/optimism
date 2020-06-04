import '../setup'

/* External Imports */
import { L2ToL1MessageReceiverContractDefinition } from '@eth-optimism/rollup-contracts'

import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

describe('L2 -> L1 Message Receiver', () => {
  let provider
  let wallet
  let l2ToL1MessageReciever
  const finalizationTime = 55

  const nonce = 0
  const callData = '0xdeadbeefee5555'
  const randomSender = '0x1234123412341234123412341234123412341234'
  const randomMessage = {
    nonce,
    ovmSender: randomSender,
    callData,
  }

  beforeEach(async () => {
    provider = createMockProvider()
    wallet = getWallets(provider)[0]
    l2ToL1MessageReciever = await deployContract(
      wallet,
      L2ToL1MessageReceiverContractDefinition,
      [wallet.address, finalizationTime]
    )
  })

  it('should allow the trusted sequencer to enqueue a message', async () => {
    await l2ToL1MessageReciever
      .enqueueL2ToL1Message(randomMessage)
      .should.emit(l2ToL1MessageReciever, 'L2ToL1MessageEnqueued')
  })
  it('should not verify message if time has not elapsed', async () => {
    await l2ToL1MessageReciever.enqueueL2ToL1Message(randomMessage)
    const verify = await l2ToL1MessageReciever.verifyL2ToL1Message(
      randomMessage,
      0
    )
    verify.should.equal(false)
  })
  it('shuld verify message once time has elapsed', async () => {
    await l2ToL1MessageReciever.enqueueL2ToL1Message(randomMessage)
    for (let i = 0; i < finalizationTime; i++) {
      await provider.send('evm_mine', [])
    }
    const verify = await l2ToL1MessageReciever.verifyL2ToL1Message(
      randomMessage,
      0
    )
    verify.should.equal(true)
  })
  it('should not verify message if time has elapsed but wrong message', async () => {
    await l2ToL1MessageReciever.enqueueL2ToL1Message(randomMessage)
    for (let i = 0; i < finalizationTime; i++) {
      await provider.send('evm_mine', [])
    }
    const wrongMessage = {
      nonce,
      ovmSender: randomSender,
      callData: '0x0101011010',
    }
    const verify = await l2ToL1MessageReciever.verifyL2ToL1Message(
      wrongMessage,
      0
    )
    verify.should.equal(false)
  })
})
