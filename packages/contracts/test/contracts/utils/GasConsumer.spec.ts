import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory } from 'ethers'
import {
  hexStrToNumber,
  numberToHexString,
  hexStrToBuf,
} from '@eth-optimism/core-utils'

/* Internal Imports */

/* Tests */
describe('GasConsumer', () => {
  let GasConsumer: ContractFactory
  let gasConsumer: Contract
  before(async () => {
    GasConsumer = await ethers.getContractFactory('GasConsumer')
    gasConsumer = await GasConsumer.deploy()
  })

  const EVM_TX_BASE_GAS_COST = 21_000

  const getTxCalldataGasCostForConsumeGas = (toConsume: number): number => {
    const expectedCalldata: Buffer = hexStrToBuf(
      GasConsumer.interface.encodeFunctionData('consumeGas', [toConsume])
    )
    const nonzeroByteCost = 16
    const zeroBytecost = 4

    let txCalldataGas = 0
    for (const [index, byte] of expectedCalldata.entries()) {
      if (byte === 0) {
        txCalldataGas += zeroBytecost
      } else {
        txCalldataGas += nonzeroByteCost
      }
    }
    return txCalldataGas
  }

  describe('Precise gas Consumption', async () => {
    for (const toConsume of [1_000, 10_000, 20_123, 100_000, 200_069]) {
      it(`should properly consume ${toConsume} gas`, async () => {
        const tx = await gasConsumer.consumeGas(toConsume)
        const receipt = await gasConsumer.provider.getTransactionReceipt(
          tx.hash
        )
        const gasUsed: number = hexStrToNumber(receipt.gasUsed._hex)

        const gasFromTxCalldata = getTxCalldataGasCostForConsumeGas(toConsume)

        gasUsed.should.eq(toConsume + gasFromTxCalldata + EVM_TX_BASE_GAS_COST)
      })
    }
  })
})
