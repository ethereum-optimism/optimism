/* Imports: External */
import { Contract, BigNumber } from 'ethers'
import { ethers } from 'hardhat'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'

export const traceToGasByOpcode = (structLogs, opcode) => {
  let gas = 0
  const opcodes = []
  for (const log of structLogs) {
    if (log.op === opcode) {
      opcodes.push(opcode)
      gas += log.gasCost
    }
  }
  return gas
}

describe('Hard forks', () => {
  let env: OptimismEnv
  let SimpleStorage: Contract
  let SelfDestruction: Contract
  let Precompiles: Contract

  before(async () => {
    env = await OptimismEnv.new()
    const Factory__SimpleStorage = await ethers.getContractFactory(
      'SimpleStorage',
      env.l2Wallet
    )
    SimpleStorage = await Factory__SimpleStorage.deploy()

    const Factory__SelfDestruction = await ethers.getContractFactory(
      'SelfDestruction',
      env.l2Wallet
    )
    SelfDestruction = await Factory__SelfDestruction.deploy()

    const Factory__Precompiles = await ethers.getContractFactory(
      'Precompiles',
      env.l2Wallet
    )
    Precompiles = await Factory__Precompiles.deploy()
  })

  describe('Berlin', () => {
    // https://eips.ethereum.org/EIPS/eip-2929
    describe('EIP-2929', () => {
      it('should update the gas schedule', async () => {
        // Get the tip height
        const tip = await env.l2Provider.getBlock('latest')

        // send a transaction to be able to trace
        const tx = await SimpleStorage.setValueNotXDomain(
          `0x${'77'.repeat(32)}`
        )
        await tx.wait()

        // Collect the traces
        const berlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          [tx.hash]
        )
        const preBerlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          [tx.hash, { overrides: { berlinBlock: tip.number * 2 } }]
        )
        expect(berlinTrace.gas).to.not.eq(preBerlinTrace.gas)

        const berlinSstoreCosts = traceToGasByOpcode(
          berlinTrace.structLogs,
          'SSTORE'
        )
        const preBerlinSstoreCosts = traceToGasByOpcode(
          preBerlinTrace.structLogs,
          'SSTORE'
        )
        expect(berlinSstoreCosts).to.not.eq(preBerlinSstoreCosts)
      })
    })

    // https://eips.ethereum.org/EIPS/eip-2565
    describe('EIP-2565', async () => {
      it('should become cheaper', async () => {
        const tip = await env.l2Provider.getBlock('latest')

        const tx = await Precompiles.expmod(64, 1, 64, { gasLimit: 5_000_000 })
        await tx.wait()

        const berlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          [tx.hash]
        )
        const preBerlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          [tx.hash, { overrides: { berlinBlock: tip.number * 2 } }]
        )
        expect(berlinTrace.gas).to.be.lt(preBerlinTrace.gas)
      })
    })
  })

  // Optimism includes EIP-3529 as part of its Berlin hardfork. It is part
  // of the London hardfork on L1. Since it is coupled to the Berlin
  // hardfork, some of its functionality cannot be directly tests via
  // integration tests since we can currently only turn on all of the Berlin
  // EIPs or none of the Berlin EIPs
  describe('Berlin Additional (L1 London)', () => {
    // https://eips.ethereum.org/EIPS/eip-3529
    describe('EIP-3529', async () => {
      const bytes32Zero = '0x' + '00'.repeat(32)
      const bytes32NonZero = '0x' + 'ff'.repeat(32)

      it('should lower the refund for storage clear', async () => {
        const tip = await env.l2Provider.getBlock('latest')

        const value = await SelfDestruction.callStatic.data()
        // It should be non zero
        expect(BigNumber.from(value).toNumber()).to.not.eq(0)

        {
          // Set the value to another non zero value
          // Going from non zero to non zero
          const tx = await SelfDestruction.setData(bytes32NonZero, {
            gasLimit: 5_000_000,
          })
          await tx.wait()

          const berlinTrace = await env.l2Provider.send(
            'debug_traceTransaction',
            [tx.hash]
          )
          const preBerlinTrace = await env.l2Provider.send(
            'debug_traceTransaction',
            [tx.hash, { overrides: { berlinBlock: tip.number * 2 } }]
          )
          // Updating a non zero value to another non zero value should not change
          expect(berlinTrace.gas).to.deep.eq(preBerlinTrace.gas)
        }

        {
          // Set the value to the zero value
          // Going from non zero to zero
          const tx = await SelfDestruction.setData(bytes32Zero, {
            gasLimit: 5_000_000,
          })
          await tx.wait()

          const berlinTrace = await env.l2Provider.send(
            'debug_traceTransaction',
            [tx.hash]
          )
          const preBerlinTrace = await env.l2Provider.send(
            'debug_traceTransaction',
            [tx.hash, { overrides: { berlinBlock: tip.number * 2 } }]
          )

          // Updating to a zero value from a non zero value should becomes
          // more expensive due to this change being coupled with EIP-2929
          expect(berlinTrace.gas).to.be.gt(preBerlinTrace.gas)
        }

        {
          // Set the value to a non zero value
          // Going from zero to non zero
          const tx = await SelfDestruction.setData(bytes32NonZero, {
            gasLimit: 5_000_000,
          })
          await tx.wait()

          const berlinTrace = await env.l2Provider.send(
            'debug_traceTransaction',
            [tx.hash]
          )
          const preBerlinTrace = await env.l2Provider.send(
            'debug_traceTransaction',
            [tx.hash, { overrides: { berlinBlock: tip.number * 2 } }]
          )

          // Updating to a zero value from a non zero value should becomes
          // more expensive due to this change being coupled with EIP-2929
          expect(berlinTrace.gas).to.be.gt(preBerlinTrace.gas)
        }
      })

      it('should remove the refund for selfdestruct', async () => {
        const tip = await env.l2Provider.getBlock('latest')

        // Send transaction with a large gas limit
        const tx = await SelfDestruction.destruct({ gasLimit: 5_000_000 })
        await tx.wait()

        const berlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          [tx.hash]
        )
        const preBerlinTrace = await env.l2Provider.send(
          'debug_traceTransaction',
          [tx.hash, { overrides: { berlinBlock: tip.number * 2 } }]
        )

        // The berlin execution should use more gas than the pre Berlin
        // execution because there is no longer a selfdestruct gas
        // refund
        expect(berlinTrace.gas).to.be.gt(preBerlinTrace.gas)
      })
    })
  })
})
