import '../setup'
import { bufferUtils } from '../../src/app'

describe('Buffer Utils tests', () => {
  describe('numberToBufferPacked', () => {
    it('works for single-byte small number', () => {
      const buf: Buffer = bufferUtils.numberToBufferPacked(2)

      buf.should.eql(Buffer.from('02', 'hex'))
    })

    it('works for single-byte big number', () => {
      const buf: Buffer = bufferUtils.numberToBufferPacked(22)

      buf.should.eql(Buffer.from('16', 'hex'))
    })

    it('works for multi-byte big number', () => {
      const buf: Buffer = bufferUtils.numberToBufferPacked(333)

      buf.should.eql(Buffer.from('014d', 'hex'))
    })
  })
})
