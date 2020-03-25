import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { abi, getLogger, add0x } from '@eth-optimism/core-utils'
import { Contract } from 'ethers'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'

/* Internal Imports */
import { DEFAULT_OPCODE_WHITELIST_MASK, GAS_LIMIT } from '../../src/app'
import {
  DEFAULT_ETHNODE_GAS_LIMIT,
  gasLimit,
  encodeMethodId,
  encodeRawArguments,
} from '../helpers'
import { fromPairs } from 'lodash'

const log = getLogger('execution-manager-storage', true)
const methodIds = fromPairs(
  ['ovmSSTORE', 'ovmSLOAD'].map((methodId) => [
    methodId,
    encodeMethodId(methodId),
  ])
)

/*********
 * TESTS *
 *********/

describe('ExecutionManager -- Storage opcodes', () => {
  const provider = createMockProvider({ gasLimit: DEFAULT_ETHNODE_GAS_LIMIT })
  const [wallet] = getWallets(provider)
  let executionManager: Contract
  // Useful constants
  const ONE_FILLED_BYTES_32 = '0x' + '11'.repeat(32)
  const TWO_FILLED_BYTES_32 = '0x' + '22'.repeat(32)

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [DEFAULT_OPCODE_WHITELIST_MASK, '0x' + '00'.repeat(20), GAS_LIMIT, true],
      { gasLimit: DEFAULT_ETHNODE_GAS_LIMIT }
    )
  })

  const sstore = async (): Promise<void> => {
    const data = add0x(
      encodeMethodId('ovmSSTORE') +
        encodeRawArguments([ONE_FILLED_BYTES_32, TWO_FILLED_BYTES_32])
    )
    // Now actually apply it to our execution manager
    const tx = await wallet.sendTransaction({
      to: executionManager.address,
      data,
      gasLimit,
    })

    const reciept = await provider.getTransactionReceipt(tx.hash)
    // Now make sure the SetStorage event was emitted
    const rawSetStorageEvent = reciept.logs[0].data
    const decodedSetStorageEvent = abi.decode(
      ['address', 'bytes32', 'bytes32'],
      rawSetStorageEvent
    )

    // Make sure we got back what we expect
    decodedSetStorageEvent[1].should.equal(ONE_FILLED_BYTES_32)
    decodedSetStorageEvent[2].should.equal(TWO_FILLED_BYTES_32)
  }

  /*
   * Test SSTORE opcode
   */
  describe('ovmSSTORE', async () => {
    it('successfully stores without throwing', async () => {
      await sstore()
    })
  })

  /*
   * Test SLOAD opcode
   */
  describe('ovmSLOAD', async () => {
    it('loads a value immediately after it is stored', async () => {
      await sstore()

      const data = add0x(
        encodeMethodId('ovmSLOAD') +
          encodeRawArguments([ONE_FILLED_BYTES_32, TWO_FILLED_BYTES_32])
      )

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit,
      })

      // It should load the value which we just set
      result.should.equal(TWO_FILLED_BYTES_32)
    })
  })
})
