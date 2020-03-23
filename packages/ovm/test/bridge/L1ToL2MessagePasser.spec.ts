import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Contract Imports */
import * as L1ToL2MessagePasser from '../../build/contracts/L1ToL2MessagePasser.json'

describe('L1 -> L2 Message Passer', () => {
  let provider
  let wallet
  let l1ToL2MessagePasser
  beforeEach(async () => {
    provider = createMockProvider()
    wallet = getWallets(provider)[0]
    l1ToL2MessagePasser = await deployContract(wallet, L1ToL2MessagePasser)
  })
  const entryPoint = '0x0000000000000000000000000000000000001234'
  const callData = '0xdeadbeefee5555'
  it('should emit the correct L1ToL2Message L1 event when an L1->L2 message is sent', async () => {
    await l1ToL2MessagePasser
      .passMessageToL2(entryPoint, callData)
      .should.emit(l1ToL2MessagePasser, 'L1ToL2Message')
      .withArgs(wallet.address, entryPoint, callData)
  })
})
