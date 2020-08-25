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
  let GasConsumerCaller: ContractFactory
  let gasConsumerCaller: Contract
  before(async () => {
    GasConsumer = await ethers.getContractFactory('GasConsumer')
    gasConsumer = await GasConsumer.deploy()
    GasConsumerCaller = await ethers.getContractFactory('GasConsumerCaller')
    gasConsumerCaller = await GasConsumerCaller.deploy()
  })

  const EVM_TX_BASE_GAS_COST = 21_000
  const RANDOM_GAS_VALUES = [4_200, 10_100, 20_123, 100_257, 1_002_769]

  const getTxCalldataGasCostForConsumeGas = (toConsume: number): number => {
    const expectedCalldata: Buffer = hexStrToBuf(
      GasConsumer.interface.encodeFunctionData('consumeGasEOA', [toConsume])
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

  describe('Precise gas consumption -- EOA entrypoint consumption', async () => {
    for (const toConsume of RANDOM_GAS_VALUES) {
      it(`should properly consume ${toConsume} gas`, async () => {
        const tx = await gasConsumer.consumeGasEOA(toConsume)
        const receipt = await gasConsumer.provider.getTransactionReceipt(
          tx.hash
        )
        const gasUsed: number = hexStrToNumber(receipt.gasUsed._hex)

        const gasFromTxCalldata = getTxCalldataGasCostForConsumeGas(toConsume)

        gasUsed.should.eq(toConsume + gasFromTxCalldata + EVM_TX_BASE_GAS_COST)
      })
    }
  })
  describe('Precise gas consumption -- internal/cross-contract consumption', async () => {
    for (const toConsume of RANDOM_GAS_VALUES) {
      it(`Should properly consume ${toConsume} gas`, async () => {
        const data = gasConsumerCaller.interface.encodeFunctionData(
          'getGasConsumedByGasConsumer',
          [gasConsumer.address, toConsume]
        )
        const tx = {
          to: gasConsumerCaller.address,
          data,
        }
        const returnedGasChange = await gasConsumerCaller.provider.call(tx)

        hexStrToNumber(returnedGasChange).should.equal(toConsume)
      })
    }
  })
})
