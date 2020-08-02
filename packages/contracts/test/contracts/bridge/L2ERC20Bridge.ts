import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, sleep, TestUtils } from '@eth-optimism/core-utils'
import { Signer, ContractFactory, Contract } from 'ethers'
import { MessageChannel } from 'worker_threads'

/* Logging */
const log = getLogger('rollup-queue', true)

/* Tests */
describe.only('L2ERC20Bridge', () => {
  const provider = ethers.provider

  let wallet: Signer
  let L2ERC20Bridge: ContractFactory
  before(async () => {
    ;[wallet] = await ethers.getSigners()
    L2ERC20Bridge = await ethers.getContractFactory('L2ERC20Bridge')
  })

  let l2ERC20Bridge: Contract
  beforeEach(async () => {
    l2ERC20Bridge = await L2ERC20Bridge.deploy(wallet.getAddress())
  })

  describe('deployNewDepositedERC20()', async () => {
    it('throws if this ERC20 already exists on L2', async () => {
      await l2ERC20Bridge.deployNewDepositedERC20('0x' + '00'.repeat(20))
      await TestUtils.assertRevertsAsync(
        'L2 ERC20 Contract for this asset already exists.',
        async () => {
          await l2ERC20Bridge.deployNewDepositedERC20('0x' + '00'.repeat(20))
        }
      )
    })
  })
})
