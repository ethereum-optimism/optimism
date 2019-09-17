import '../../setup'

/* External Imports */
import debug from 'debug'
const log = debug('test:info:rollup-encoding')

/* Internal Imports */
import { AbiTransferTx, AbiSwapTx, AbiSignedTx } from '../../../src/app'

describe('RollupEncoding', () => {
  it('should encoded & decode AbiTransferTx without throwing', async () => {
    const address = '0x' + '31'.repeat(20)
    const tx = new AbiTransferTx(address, address, 1, 15)
    log(tx.encoded)
    log(AbiTransferTx.from(tx.encoded))
  })

  it('should encoded & decode AbiSwapTx without throwing', async () => {
    const address = '0x' + '31'.repeat(20)
    const tx = new AbiSwapTx(address, 1, 15, 4, +new Date())
    log(tx.encoded)
    log(AbiSwapTx.from(tx.encoded))
  })

  it('should encoded & decode AbiSignedTx without throwing', async () => {
    const address = '0x' + '31'.repeat(20)
    const transferTx = new AbiTransferTx(address, address, 1, 15)
    const swapTx = new AbiSwapTx(address, 1, 15, 4, +new Date())
    const transferSignedTx = new AbiSignedTx('0x1234', transferTx)
    const swapSignedTx = new AbiSignedTx('0x1234', swapTx)
    // Check transfer
    log(transferSignedTx.encoded)
    log(AbiSignedTx.from(transferSignedTx.encoded))
    // Check swap
    log(swapSignedTx.encoded)
    log(AbiSignedTx.from(swapSignedTx.encoded))
  })
})
