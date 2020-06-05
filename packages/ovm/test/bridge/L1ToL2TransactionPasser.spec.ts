import '../setup'

/* External Imports */
import { L1ToL2TransactionPasserContractDefinition } from '@eth-optimism/rollup-contracts'

import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Contract Imports */
describe('L1 -> L2 Transaction Passer', () => {
  let provider
  let wallet
  let l1ToL2TransactionPasser
  beforeEach(async () => {
    provider = createMockProvider()
    wallet = getWallets(provider)[0]
    l1ToL2TransactionPasser = await deployContract(
      wallet,
      L1ToL2TransactionPasserContractDefinition
    )
  })
  const entryPoint = '0x0000000000000000000000000000000000001234'
  const callData = '0xdeadbeefee5555'
  it('should emit the correct L1ToL2Transaction L1 event when an L1->L2 tx is sent', async () => {
    await l1ToL2TransactionPasser
      .passTransactionToL2(entryPoint, callData)
      .should.emit(l1ToL2TransactionPasser, 'L1ToL2Transaction')
      .withArgs(0, wallet.address, entryPoint, callData)
  })
})
