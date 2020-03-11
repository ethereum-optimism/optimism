import '../setup'

/* External Imports */
import chai = require('chai')
import {
  createMockProvider,
  deployContract,
  getWallets,
  solidity,
} from 'ethereum-waffle'

chai.use(solidity)

/* Contract Imports */
import * as RollupMessagePasser from '../../build/RollupMessagePasser.json'

describe('Rollup Message Passer', () => {
  let provider
  let wallet
  let rollupMessagePasser
  beforeEach(async () => {
    provider = createMockProvider()
    wallet = getWallets(provider)[0]
    rollupMessagePasser = await deployContract(wallet, RollupMessagePasser)
  })
  const entryPoint = '0x0000000000000000000000000000000000001234'
  const callData = '0xdeadbeefee5555'
  it('should emit the correct L1ToL2Message L1 event when an L1->L2 message is sent', async () => {
    await rollupMessagePasser
      .passMessageToRollup(entryPoint, callData)
      .should.emit(rollupMessagePasser, 'L1ToL2Message')
      .withArgs(wallet.address, entryPoint, callData)
  })
})
