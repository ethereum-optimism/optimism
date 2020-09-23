import '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { abi, getLogger, add0x, NULL_ADDRESS } from '@eth-optimism/core-utils'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  GAS_LIMIT,
  encodeMethodId,
  encodeRawArguments,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  getDefaultGasMeterParams,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-storage', true)

/* Tests */
describe('ExecutionManager -- Storage opcodes', () => {
  const provider = ethers.provider
  const ONE_FILLED_BYTES_32 = '0x' + '11'.repeat(32)
  const TWO_FILLED_BYTES_32 = '0x' + '22'.repeat(32)

  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let ExecutionManager: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
  })

  let executionManager: Contract
  beforeEach(async () => {
    executionManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: ExecutionManager,
        params: [
          resolver.addressResolver.address,
          NULL_ADDRESS,
          getDefaultGasMeterParams(),
        ],
      }
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
      gasLimit: GAS_LIMIT,
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
    it.skip('successfully stores without throwing', async () => {
      await sstore()
    })
  })

  /*
   * Test SLOAD opcode
   */
  describe('ovmSLOAD', async () => {
    it.skip('loads a value immediately after it is stored', async () => {
      // Skipping because this uses events to verify success

      await sstore()

      const data = add0x(
        encodeMethodId('ovmSLOAD') +
          encodeRawArguments([ONE_FILLED_BYTES_32, TWO_FILLED_BYTES_32])
      )

      // Now actually apply it to our execution manager
      const result = await executionManager.provider.call({
        to: executionManager.address,
        data,
        gasLimit: GAS_LIMIT,
      })

      // It should load the value which we just set
      result.should.equal(TWO_FILLED_BYTES_32)
    })
  })
})
