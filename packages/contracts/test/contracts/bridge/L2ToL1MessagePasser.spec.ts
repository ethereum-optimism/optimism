import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'
import { MessageChannel } from 'worker_threads'
import { deployAndRegister } from 'src'
import { indexOf } from 'lodash'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Tests */
describe.only('RealL2ToL1MessagePasser', () => {
  const provider = ethers.provider
  let wallet: Signer
  let L2ToL1MessagePasser: ContractFactory

  before(async () => {
    ;[wallet] = await ethers.getSigners()
    L2ToL1MessagePasser = await ethers.getContractFactory(
      'RealL2ToL1MessagePasser'
    )
  })

  let l2ToL1MessagePasser: Contract
  beforeEach(async () => {
    l2ToL1MessagePasser = await L2ToL1MessagePasser.deploy()
  })

  describe('passMessageToL1()', async () => {
    it('increments index and resets with new block', async () => {
      const startingIndex = await l2ToL1MessagePasser.index()
      await l2ToL1MessagePasser.passMessageToL1(
        '0x' + '00'.repeat(32),
        '0x' + '00'.repeat(20)
      )
      //call it again, should re-set then re-increment to 1
      await l2ToL1MessagePasser.passMessageToL1(
        '0x' + '00'.repeat(32),
        '0x' + '00'.repeat(20)
      )
      const newIndex = await l2ToL1MessagePasser.index()
      newIndex.should.equal(startingIndex + 1)
    })
  })
})
