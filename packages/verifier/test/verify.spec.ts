import './setup'

/* Internal Imports */
import { encodeParams } from '../src/abi'
import { validStateTransition } from '../src/verify'
import { PREIMAGE_BYTECODE } from './constants'

describe('Validation', () => {
  describe('validStateTransition', () => {
    it('should correctly accept a valid state transition', async () => {
      const preimage = '0x' + Buffer.from('hello').toString('hex')
      const hash =
        '0x1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8'
      const encodedData = encodeParams(['bytes32'], [hash])
      const oldState = encodeParams(['bytes'], [encodedData])
      const newState = encodeParams(['bytes'], ['0x00'])
      const witness = encodeParams(['bytes'], [preimage])

      const valid = await validStateTransition(
        oldState,
        newState,
        witness,
        PREIMAGE_BYTECODE
      )
      valid.should.be.true
    })

    it('should correctly reject an invalid state transition', async () => {
      const preimage = '0x' + Buffer.from('goodbye').toString('hex')
      const hash =
        '0x1c8aff950685c2ed4bc3174f3472287b56d9517b9c948127319a09a7a36deac8'
      const encodedData = encodeParams(['bytes32'], [hash])
      const oldState = encodeParams(['bytes'], [encodedData])
      const newState = encodeParams(['bytes'], ['0x00'])
      const witness = encodeParams(['bytes'], [preimage])

      const valid = await validStateTransition(
        oldState,
        newState,
        witness,
        PREIMAGE_BYTECODE
      )
      valid.should.be.false
    })
  })
})
