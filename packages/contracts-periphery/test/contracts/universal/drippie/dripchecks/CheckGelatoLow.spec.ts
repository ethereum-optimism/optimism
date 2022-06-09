import { Contract } from 'ethers'
import { smock, FakeContract } from '@defi-wonderland/smock'

import { expect } from '../../../../setup'
import { deploy } from '../../../../helpers'
import { encodeDripCheckParams } from '../../../../../src'

describe('CheckGelatoLow', () => {
  const RECIPIENT = '0x' + '11'.repeat(20)
  const THRESHOLD = 100

  let CheckGelatoLow: Contract
  let FakeGelatoTresury: FakeContract<Contract>
  before(async () => {
    CheckGelatoLow = await deploy('CheckGelatoLow')
    FakeGelatoTresury = await smock.fake('IGelatoTreasury')
  })

  describe('check', () => {
    it('should return true when balance is below threshold', async () => {
      FakeGelatoTresury.userTokenBalance.returns(THRESHOLD - 1)

      expect(
        await CheckGelatoLow.check(
          encodeDripCheckParams(CheckGelatoLow.interface, {
            treasury: FakeGelatoTresury.address,
            threshold: THRESHOLD,
            recipient: RECIPIENT,
          })
        )
      ).to.equal(true)
    })

    it('should return false when balance is above threshold', async () => {
      FakeGelatoTresury.userTokenBalance.returns(THRESHOLD + 1)

      expect(
        await CheckGelatoLow.check(
          encodeDripCheckParams(CheckGelatoLow.interface, {
            treasury: FakeGelatoTresury.address,
            threshold: THRESHOLD,
            recipient: RECIPIENT,
          })
        )
      ).to.equal(false)
    })
  })
})
