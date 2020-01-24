import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { abi, add0x, getLogger, remove0x } from '@pigi/core-utils'
import * as ethereumjsAbi from 'ethereumjs-abi'
import { ContractFactory } from 'ethers'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleStorage from '../../build/contracts/SimpleStorage.json'
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'

const log = getLogger('execution-manager-create', true)

/*********
 * TESTS *
 *********/

describe('ExecutionManager -- Create opcodes', () => {
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

  /*
   * Test CREATE opcode
   */
  describe('ovmCREATE', async () => {
    it('does not throw when passed bytecode', async () => {
      const deployTx = new ContractFactory(
        SimpleStorage.abi,
        SimpleStorage.bytecode
      ).getDeployTransaction(executionManager.address)

      const methodId: string = ethereumjsAbi
        .methodID('ovmCREATE', [])
        .toString('hex')

      const data = `0x${methodId}${remove0x(deployTx.data)}`

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: 6_700_000,
      })

      log.debug(`Result: [${result}]`)

      result.length.should.be.greaterThan(2, 'Should not just be 0x')
    })
  })
})
