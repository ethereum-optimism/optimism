import { expect } from '../../common/setup'

/* External Imports */
import { deployContract } from 'ethereum-waffle-v2'
import { Contract, Wallet } from 'ethers-v4'

/* Internal Imports */
import { waffleV2 } from '../../../src/waffle/waffle-v2'

/* Contract Imports */
import * as TimestampCheckerContract from '../../temp/build/waffle/TimestampChecker.json'

const overrides = {
  gasLimit: 100000000,
}

describe('Timestamp Manipulation Support', () => {
  let provider: any
  let wallet: Wallet
  let timestampChecker: Contract
  beforeEach(async () => {
    provider = new waffleV2.MockProvider(overrides)
    ;[wallet] = provider.getWallets()
    timestampChecker = await deployContract(
      wallet,
      TimestampCheckerContract,
      [],
      overrides
    )
  })

  it('should retrieve initial timestamp correctly', async () => {
    const timestamp = await timestampChecker.getTimestamp()

    expect(timestamp.toNumber()).to.equal(
      0,
      'Initial timestamp was not set to zero'
    )
  })

  it('should retrieve the block timestamp correctly after modifying with evm_mine', async () => {
    const beforeTimestamp = (await timestampChecker.blockTimestamp()).toNumber()
    await provider.rpc('evm_mine', [beforeTimestamp + 10])
    const afterTimestamp = (await timestampChecker.blockTimestamp()).toNumber()

    expect(beforeTimestamp + 10).to.equal(
      afterTimestamp,
      'Block timestamp was incorrect'
    )
  })
})
