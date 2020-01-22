import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { abi, getLogger, remove0x } from '@pigi/core-utils'
import * as ethereumjsAbi from 'ethereumjs-abi'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'

const log = getLogger('execution-manager-storage', true)

/*********
 * TESTS *
 *********/

describe('ExecutionManager -- Storage opcodes', () => {
  const provider = createMockProvider()
  const [wallet] = getWallets(provider)
  let executionManager
  let contractAddressGenerator
  let rlpEncode
  // Useful constants
  const ONE_FILLED_BYTES_32 = '0x' + '11'.repeat(32)
  const TWO_FILLED_BYTES_32 = '0x' + '22'.repeat(32)

  /* Link libraries before tests */
  before(async () => {
    rlpEncode = await deployContract(wallet, RLPEncode, [], {
      gasLimit: 6700000,
    })
    contractAddressGenerator = await deployContract(
      wallet,
      ContractAddressGenerator,
      [rlpEncode.address],
      {
        gasLimit: 6700000,
      }
    )
  })

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    executionManager = await deployContract(
      wallet,
      ExecutionManager,
      [
        '0x' + '00'.repeat(20),
        contractAddressGenerator.address,
        '0x' + '00'.repeat(20),
      ],
      {
        gasLimit: 6700000,
      }
    )
  })

  const sstore = async (): Promise<void> => {
    const methodId: string = ethereumjsAbi
      .methodID('ovmSSTORE', [])
      .toString('hex')

    const data = `0x${methodId}${remove0x(ONE_FILLED_BYTES_32)}${remove0x(
      TWO_FILLED_BYTES_32
    )}`

    // Now actually apply it to our execution manager
    const tx = await wallet.sendTransaction({
      to: executionManager.address,
      data,
      gasLimit: 6_700_000,
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

      const methodId: string = ethereumjsAbi
        .methodID('ovmSLOAD', [])
        .toString('hex')

      const data = `0x${methodId}${remove0x(ONE_FILLED_BYTES_32)}`

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      // It should load the value which we just set
      result.should.equal(TWO_FILLED_BYTES_32)
    })
  })
})
