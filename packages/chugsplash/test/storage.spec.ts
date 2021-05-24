import { expect } from './setup'

/* Imports: External */
import hre from 'hardhat'
import { Contract } from 'ethers'
import { isObject, toPlainObject } from 'lodash'

/* Imports: Internal */
import {
  computeStorageSlots,
  getStorageLayout,
  SolidityStorageLayout,
} from '../src'

describe('ChugSplash storage layout parsing', () => {
  const ethers = (hre as any).ethers // as Ethers (???)

  let layout: SolidityStorageLayout
  before(async () => {
    layout = await getStorageLayout('Helper_StorageHelper')
  })

  let Helper_StorageHelper: Contract
  beforeEach(async () => {
    Helper_StorageHelper = await (
      await ethers.getContractFactory('Helper_StorageHelper')
    ).deploy()
  })

  const computeAndVerifyStorageSlots = async (
    variables: any
  ): Promise<void> => {
    const slots = computeStorageSlots(layout, variables)
    for (const slot of slots) {
      await Helper_StorageHelper.setStorage(slot.key, slot.val)
    }

    for (const variable of Object.keys(variables)) {
      const valA = await Helper_StorageHelper[variable]()
      const valB = variables[variable]
      if (Array.isArray(valA) && isObject(valB)) {
        expect(toPlainObject(valA)).to.deep.include(valB)
      } else {
        expect(valA).to.equal(valB)
      }
    }
  }

  describe('computeStorageSlots', () => {
    it('compute slots for uint8', async () => {
      await computeAndVerifyStorageSlots({
        _uint8: 123,
      })
    })

    it('compute slots for uint64', async () => {
      await computeAndVerifyStorageSlots({
        _uint64: 1234,
      })
    })

    it('compute slots for uint256', async () => {
      await computeAndVerifyStorageSlots({
        _uint256: 12345,
      })
    })

    it('compute slots for bytes1', async () => {
      await computeAndVerifyStorageSlots({
        _bytes1: '0x11',
      })
    })

    it('compute slots for bytes8', async () => {
      await computeAndVerifyStorageSlots({
        _bytes8: '0x1212121212121212',
      })
    })

    it('compute slots for bytes32', async () => {
      await computeAndVerifyStorageSlots({
        _bytes32:
          '0x2222222222222222222222222222222222222222222222222222222222222222',
      })
    })

    it('compute slots for bool', async () => {
      await computeAndVerifyStorageSlots({
        _bool: true,
      })
    })

    it('compute slots for address', async () => {
      await computeAndVerifyStorageSlots({
        _address: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
      })
    })

    it('compute slots for bytes (<32 bytes long)', async () => {
      await computeAndVerifyStorageSlots({
        _bytes:
          '0x12121212121212121212121212121212121212121212121212121212121212', // only 31 bytes
      })
    })

    it('compute slots for string (<32 bytes long)', async () => {
      await computeAndVerifyStorageSlots({
        _string: 'hello i am a string', // 19 bytes
      })
    })

    it('compute slots for a simple (complete) struct', async () => {
      await computeAndVerifyStorageSlots({
        _struct: {
          _structUint256: ethers.BigNumber.from(1234),
          _structAddress: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
          _structBytes32:
            '0x1212121212121212121212121212121212121212121212121212121212121212',
        },
      })
    })

    it('compute slots for packed variables', async () => {
      await computeAndVerifyStorageSlots({
        _packedAddress: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
        _packedBool: true,
        _packedBytes11: '0x1212121212121212121212',
      })
    })

    it('compute slots for packed variables (2)', async () => {
      await computeAndVerifyStorageSlots({
        _otherPackedBytes11: '0x1212121212121212121212',
        _otherPackedBool: true,
        _otherPackedAddress: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
      })
    })

    describe('unsupported types', () => {
      it('should not support mappings', () => {
        expect(() => {
          computeStorageSlots(layout, {
            _uint256ToUint256Map: {
              1234: 5678,
            },
          })
        }).to.throw('mapping types not yet supported')
      })

      it('should not support arrays', () => {
        expect(() => {
          computeStorageSlots(layout, {
            _uint256Array: [1234, 5678],
          })
        }).to.throw('array types not yet supported')
      })

      it('should not support bytes > 31 bytes long', () => {
        expect(() => {
          computeStorageSlots(layout, {
            _bytes: '0x' + '22'.repeat(64),
          })
        }).to.throw('large strings (>31 bytes) not supported')
      })

      it('should not support strings > 31 bytes long', () => {
        expect(() => {
          computeStorageSlots(layout, {
            _string: 'hello'.repeat(32),
          })
        }).to.throw('large strings (>31 bytes) not supported')
      })
    })
  })
})
