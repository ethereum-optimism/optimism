import { Contract } from 'ethers'
import { ethers } from 'hardhat'

import { OptimismEnv } from '../shared/env'
import { expect } from '../shared/setup'
import { traceToGasByOpcode } from '../hardfork.spec'
import { envConfig } from '../shared/utils'

describe('Nightly', () => {
  before(async function () {
    if (!envConfig.RUN_NIGHTLY_TESTS) {
      this.skip()
    }
  })

  describe('Berlin Hardfork', () => {
    let env: OptimismEnv
    let SimpleStorage: Contract
    let Precompiles: Contract

    before(async () => {
      env = await OptimismEnv.new()
      SimpleStorage = await ethers.getContractAt(
        'SimpleStorage',
        '0xE08fFE40748367ddc29B5A154331C73B7FCC13bD',
        env.l2Wallet
      )

      Precompiles = await ethers.getContractAt(
        'Precompiles',
        '0x32E8Fbfd0C0bd1117112b249e997C27b0EC7cba2',
        env.l2Wallet
      )
    })

    describe('EIP-2929', () => {
      it('should update the gas schedule', async () => {
        const tx = await SimpleStorage.setValueNotXDomain(
          `0x${'77'.repeat(32)}`
        )
        await tx.wait()

        const berlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          [tx.hash]
        )
        const preBerlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          ['0x2bb346f53544c5711502fbcbd1d78dc4fb61ca5f9390b9d6d67f1a3a77de7c39']
        )

        const berlinSstoreCosts = traceToGasByOpcode(
          berlinTrace.structLogs,
          'SSTORE'
        )
        const preBerlinSstoreCosts = traceToGasByOpcode(
          preBerlinTrace.structLogs,
          'SSTORE'
        )
        expect(preBerlinSstoreCosts).to.eq(80000)
        expect(berlinSstoreCosts).to.eq(5300)
      })
    })

    describe('EIP-2565', () => {
      it('should become cheaper', async () => {
        const tx = await Precompiles.expmod(64, 1, 64, { gasLimit: 5_000_000 })
        await tx.wait()

        const berlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          [tx.hash]
        )
        const preBerlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          ['0x7ba7d273449b0062448fe5e7426bb169a032ce189d0e3781eb21079e85c2d7d5']
        )
        expect(berlinTrace.gas).to.be.lt(preBerlinTrace.gas)
      })
    })

    describe('Berlin Additional (L1 London)', () => {
      describe('EIP-3529', () => {
        it('should remove the refund for selfdestruct', async () => {
          const Factory__SelfDestruction = await ethers.getContractFactory(
            'SelfDestruction',
            env.l2Wallet
          )

          const SelfDestruction = await Factory__SelfDestruction.deploy()
          const tx = await SelfDestruction.destruct({ gasLimit: 5_000_000 })
          await tx.wait()

          const berlinTrace = await env.l2Provider.send(
            'debug_traceTransaction',
            [tx.hash]
          )
          const preBerlinTrace = await env.l2Provider.send(
            'debug_traceTransaction',
            [
              '0x948667349f00e996d9267e5c30d72fe7202a0ecdb88bab191e9a022bba6e4cb3',
            ]
          )
          expect(berlinTrace.gas).to.be.gt(preBerlinTrace.gas)
        })
      })
    })
  })
})
