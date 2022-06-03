import hre from 'hardhat'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { Contract } from 'ethers'
import { toRpcHexString } from '@eth-optimism/core-utils'

import { expect } from '../../../setup'
import { deploy } from '../../../helpers'

describe('Drippie', () => {
  const DEFAULT_DRIP_NAME = 'drippity drip drip'
  const DEFAULT_DRIP_CONFIG = {
    interval: hre.ethers.BigNumber.from(100),
    dripcheck: '', // Gets added in setup
    checkparams: '0x',
    actions: [
      {
        target: '0x' + '11'.repeat(20),
        data: '0x',
        value: hre.ethers.BigNumber.from(1),
      },
    ],
  }

  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before('signer setup', async () => {
    ;[signer1, signer2] = await hre.ethers.getSigners()
  })

  before('deploy default dripcheck', async () => {
    DEFAULT_DRIP_CONFIG.dripcheck = (await deploy('CheckTrue')).address
  })

  let SimpleStorage: Contract
  let Drippie: Contract
  beforeEach('deploy contracts', async () => {
    SimpleStorage = await deploy('SimpleStorage')
    Drippie = await deploy('Drippie', {
      signer: signer1,
      args: [signer1.address],
    })
  })

  beforeEach('balance setup', async () => {
    await hre.ethers.provider.send('hardhat_setBalance', [
      Drippie.address,
      toRpcHexString(DEFAULT_DRIP_CONFIG.actions[0].value.mul(100000)),
    ])
    await hre.ethers.provider.send('hardhat_setBalance', [
      DEFAULT_DRIP_CONFIG.actions[0].target,
      '0x0',
    ])
  })

  describe('create', () => {
    describe('when called by authorized address', () => {
      it('should create a drip with the given name', async () => {
        await expect(
          Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)
        ).to.emit(Drippie, 'DripCreated')

        const drip = await Drippie.drips(DEFAULT_DRIP_NAME)
        expect(drip.status).to.equal(2) // PAUSED
        expect(drip.last).to.deep.equal(hre.ethers.BigNumber.from(0))
        expect(drip.config.interval).to.deep.equal(DEFAULT_DRIP_CONFIG.interval)
        expect(drip.config.dripcheck).to.deep.equal(
          DEFAULT_DRIP_CONFIG.dripcheck
        )
        expect(drip.config.checkparams).to.deep.equal(
          DEFAULT_DRIP_CONFIG.checkparams
        )
        expect(drip.config.actions[0][0]).to.deep.equal(
          DEFAULT_DRIP_CONFIG.actions[0].target
        )
        expect(drip.config.actions[0][1]).to.deep.equal(
          DEFAULT_DRIP_CONFIG.actions[0].data
        )
        expect(drip.config.actions[0][2]).to.deep.equal(
          DEFAULT_DRIP_CONFIG.actions[0].value
        )
      })

      it('should not be able to create the same drip twice', async () => {
        await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)

        await expect(
          Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)
        ).to.be.revertedWith('Drippie: drip with that name already exists')
      })
    })

    describe('when called by not authorized address', () => {
      it('should revert', async () => {
        await expect(
          Drippie.connect(signer2).create(
            DEFAULT_DRIP_NAME,
            DEFAULT_DRIP_CONFIG
          )
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })

  describe('status', () => {
    describe('when called by authorized address', () => {
      it('should allow setting between PAUSED and ACTIVE', async () => {
        await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)

        expect((await Drippie.drips(DEFAULT_DRIP_NAME)).status).to.equal(2) // PAUSED

        await Drippie.status(DEFAULT_DRIP_NAME, 1) // ACTIVE

        expect((await Drippie.drips(DEFAULT_DRIP_NAME)).status).to.equal(1) // ACTIVE

        await Drippie.status(DEFAULT_DRIP_NAME, 2) // PAUSED

        expect((await Drippie.drips(DEFAULT_DRIP_NAME)).status).to.equal(2) // PAUSED
      })

      it('should not allow setting status to NONE', async () => {
        await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)

        await expect(Drippie.status(DEFAULT_DRIP_NAME, 0)).to.be.revertedWith(
          'Drippie: drip status can never be set back to NONE after creation'
        )
      })

      it('should not allow setting status to same status as before', async () => {
        await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)

        await expect(Drippie.status(DEFAULT_DRIP_NAME, 2)).to.be.revertedWith(
          'Drippie: cannot set drip status to same status as before'
        )
      })

      it('should allow setting status to ARCHIVED if PAUSED', async () => {
        await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)

        await Drippie.status(DEFAULT_DRIP_NAME, 3) // ARCHIVED

        expect((await Drippie.drips(DEFAULT_DRIP_NAME)).status).to.equal(3) // ARCHIVED
      })

      it('should not allow setting status to ARCHIVED if ACTIVE', async () => {
        await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)

        await Drippie.status(DEFAULT_DRIP_NAME, 1) // ACTIVE

        await expect(Drippie.status(DEFAULT_DRIP_NAME, 3)).to.be.revertedWith(
          'Drippie: drip must be paused to be archived'
        )
      })

      it('should not allow setting status to PAUSED if ARCHIVED', async () => {
        await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)

        await Drippie.status(DEFAULT_DRIP_NAME, 3) // ARCHIVED

        await expect(Drippie.status(DEFAULT_DRIP_NAME, 2)).to.be.revertedWith(
          'Drippie: drip with that name has been archived'
        )
      })

      it('should not allow setting status to ACTIVE if ARCHIVED', async () => {
        await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)

        await Drippie.status(DEFAULT_DRIP_NAME, 3) // ARCHIVED

        await expect(Drippie.status(DEFAULT_DRIP_NAME, 1)).to.be.revertedWith(
          'Drippie: drip with that name has been archived'
        )
      })

      it('should revert if the drip does not exist yet', async () => {
        await expect(Drippie.status(DEFAULT_DRIP_NAME, 1)).to.be.revertedWith(
          'Drippie: drip with that name does not exist'
        )
      })
    })

    describe('when called by not authorized address', () => {
      it('should revert', async () => {
        await expect(
          Drippie.connect(signer2).status(DEFAULT_DRIP_NAME, 1)
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })

  describe('drip', () => {
    it('should drip the amount', async () => {
      await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)
      await Drippie.status(DEFAULT_DRIP_NAME, 1) // ACTIVE

      await expect(Drippie.drip(DEFAULT_DRIP_NAME)).to.emit(
        Drippie,
        'DripExecuted'
      )

      expect(
        await signer1.provider.getBalance(DEFAULT_DRIP_CONFIG.actions[0].target)
      ).to.equal(DEFAULT_DRIP_CONFIG.actions[0].value)
    })

    it('should be able to trigger one function', async () => {
      await Drippie.create(DEFAULT_DRIP_NAME, {
        ...DEFAULT_DRIP_CONFIG,
        actions: [
          {
            target: SimpleStorage.address,
            data: SimpleStorage.interface.encodeFunctionData('set', [
              '0x' + '33'.repeat(32),
              '0x' + '44'.repeat(32),
            ]),
            value: hre.ethers.BigNumber.from(0),
          },
        ],
      })

      await Drippie.status(DEFAULT_DRIP_NAME, 1) // ACTIVE
      await Drippie.drip(DEFAULT_DRIP_NAME)

      expect(await SimpleStorage.get('0x' + '33'.repeat(32))).to.equal(
        '0x' + '44'.repeat(32)
      )
    })

    it('should be able to trigger two functions', async () => {
      await Drippie.create(DEFAULT_DRIP_NAME, {
        ...DEFAULT_DRIP_CONFIG,
        actions: [
          {
            target: SimpleStorage.address,
            data: SimpleStorage.interface.encodeFunctionData('set', [
              '0x' + '33'.repeat(32),
              '0x' + '44'.repeat(32),
            ]),
            value: hre.ethers.BigNumber.from(0),
          },
          {
            target: SimpleStorage.address,
            data: SimpleStorage.interface.encodeFunctionData('set', [
              '0x' + '44'.repeat(32),
              '0x' + '55'.repeat(32),
            ]),
            value: hre.ethers.BigNumber.from(0),
          },
        ],
      })

      await Drippie.status(DEFAULT_DRIP_NAME, 1) // ACTIVE
      await Drippie.drip(DEFAULT_DRIP_NAME)

      expect(await SimpleStorage.get('0x' + '33'.repeat(32))).to.equal(
        '0x' + '44'.repeat(32)
      )
      expect(await SimpleStorage.get('0x' + '44'.repeat(32))).to.equal(
        '0x' + '55'.repeat(32)
      )
    })

    it('should revert if dripping twice in one interval', async () => {
      await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)
      await Drippie.status(DEFAULT_DRIP_NAME, 1) // ACTIVE
      await Drippie.drip(DEFAULT_DRIP_NAME)

      await expect(Drippie.drip(DEFAULT_DRIP_NAME)).to.be.revertedWith(
        'Drippie: drip interval has not elapsed'
      )

      await hre.ethers.provider.send('evm_increaseTime', [
        DEFAULT_DRIP_CONFIG.interval.add(1).toHexString(),
      ])

      await expect(Drippie.drip(DEFAULT_DRIP_NAME)).to.not.be.reverted
    })

    it('should revert when the drip does not exist', async () => {
      await expect(Drippie.drip(DEFAULT_DRIP_NAME)).to.be.revertedWith(
        'Drippie: selected drip does not exist or is not currently active'
      )
    })

    it('should revert when the drip is not active', async () => {
      await Drippie.create(DEFAULT_DRIP_NAME, DEFAULT_DRIP_CONFIG)

      await expect(Drippie.drip(DEFAULT_DRIP_NAME)).to.be.revertedWith(
        'Drippie: selected drip does not exist or is not currently active'
      )
    })
  })
})
