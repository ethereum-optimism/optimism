import { expect } from '../setup'

/* Imports: External */
import { ethers } from 'ethers'

/* Imports: Internal */
import { makeRelayTransactionData } from '../../src/relay-tx'

describe('relay transaction generation functions', () => {
  describe('makeRelayTransactionData', () => {
    it('should do the thing', async () => {
      const result = await makeRelayTransactionData(
        'https://mainnet.infura.io/v3/c60b0bb42f8a4c6481ecd229eddaca27',
        'https://mainnet.optimism.io',
        '0x6786EB419547a4902d285F70c6acDbC9AefAdB6F',
        '0x4200000000000000000000000000000000000007',
        '0x031b49156168b8d587a1bd37b598e907303304ae33bcdb015996e2f427a3aef0'
      )
      console.log(result)
    })
  })
})
