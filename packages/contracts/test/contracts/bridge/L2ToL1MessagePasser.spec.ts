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
describe.only('L2ToL1MessagePasser', () => {
  const provider = ethers.provider
  let wallet: Signer
  let L2ToL1MessagePasser: ContractFactory

  before(async () => {
    ;[wallet] = await ethers.getSigners()
    L2ToL1MessagePasser = await ethers.getContractFactory('L2ToL1MessagePasser')
  })

  let l2ToL1MessagePasser: Contract
  beforeEach(async () => {
    console.log(L2ToL1MessagePasser)
    l2ToL1MessagePasser = await L2ToL1MessagePasser.deploy()
  })

  describe('passMessageToL1()', async () => {
    it('resets index per block', async () => {
        // await l2ToL1MessagePasser.passMessageToL1(
        //   '0x' + '00'.repeat(32),
        //   '0x' + '00'.repeat(20)
        // )
        //l2ToL1MessagePasser.index().should.equal(1)
    })
  })
})