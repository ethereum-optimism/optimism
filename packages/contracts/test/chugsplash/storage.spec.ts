import { expect } from '../setup'

import hre from 'hardhat'
import {
  computeStorageSlots,
  getStorageLayout,
  SolidityStorageLayout,
} from '../../src'

describe('ChugSplash storage layout parsing', () => {
  let layout: SolidityStorageLayout
  before(async () => {
    layout = await getStorageLayout(hre, 'Helper_StorageHelper')
  })

  describe('computeStorageSlots', () => {
    it('compute slots for uint8', () => {
      expect(
        computeStorageSlots(layout, {
          _uint8: 123,
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000000',
          val:
            '0x000000000000000000000000000000000000000000000000000000000000007b',
        },
      ])
    })

    it('compute slots for uint64', () => {
      expect(
        computeStorageSlots(layout, {
          _uint64: 1234,
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000002',
          val:
            '0x00000000000000000000000000000000000000000000000000000000000004d2',
        },
      ])
    })

    it('compute slots for uint256', () => {
      expect(
        computeStorageSlots(layout, {
          _uint256: 12345,
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000004',
          val:
            '0x0000000000000000000000000000000000000000000000000000000000003039',
        },
      ])
    })

    it('compute slots for bytes1', () => {
      expect(
        computeStorageSlots(layout, {
          _bytes1: '0x11',
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000006',
          val:
            '0x0000000000000000000000000000000000000000000000000000000000000011',
        },
      ])
    })

    it('compute slots for bytes8', () => {
      expect(
        computeStorageSlots(layout, {
          _bytes8: '0x1212121212121212',
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000008',
          val:
            '0x0000000000000000000000000000000000000000000000001212121212121212',
        },
      ])
    })

    it('compute slots for bytes32', () => {
      expect(
        computeStorageSlots(layout, {
          _bytes32:
            '0x2222222222222222222222222222222222222222222222222222222222222222',
        })
      ).to.deep.equal([
        {
          key:
            '0x000000000000000000000000000000000000000000000000000000000000000a',
          val:
            '0x2222222222222222222222222222222222222222222222222222222222222222',
        },
      ])
    })

    it('compute slots for bool', () => {
      expect(
        computeStorageSlots(layout, {
          _bool: true,
        })
      ).to.deep.equal([
        {
          key:
            '0x000000000000000000000000000000000000000000000000000000000000000c',
          val:
            '0x0000000000000000000000000000000000000000000000000000000000000001',
        },
      ])
    })

    it('compute slots for address', () => {
      expect(
        computeStorageSlots(layout, {
          _address: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
        })
      ).to.deep.equal([
        {
          key:
            '0x000000000000000000000000000000000000000000000000000000000000000e',
          val:
            '0x0000000000000000000000005a0b54d5dc17e0aadc383d2db43b0a0d3e029c4c',
        },
      ])
    })

    it('compute slots for bytes (<32 bytes long)', () => {
      expect(
        computeStorageSlots(layout, {
          _bytes:
            '0x12121212121212121212121212121212121212121212121212121212121212', // only 31 bytes
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000010',
          val:
            '0x121212121212121212121212121212121212121212121212121212121212123e', // last byte contains byte length * 2
        },
      ])
    })

    it('compute slots for string (<32 bytes long)', () => {
      expect(
        computeStorageSlots(layout, {
          _string: 'hello i am a string', // 19 bytes
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000011',
          val:
            '0x68656c6c6f206920616d206120737472696e6700000000000000000000000026', // 19 * 2 = 38 = 0x26
        },
      ])
    })

    it('compute slots for a simple (complete) struct', () => {
      expect(
        computeStorageSlots(layout, {
          _struct: {
            _structUint256: 1234,
            _structAddress: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
            _structBytes32:
              '0x1212121212121212121212121212121212121212121212121212121212121212',
          },
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000012',
          val:
            '0x00000000000000000000000000000000000000000000000000000000000004d2',
        },
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000013',
          val:
            '0x0000000000000000000000005a0b54d5dc17e0aadc383d2db43b0a0d3e029c4c',
        },
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000014',
          val:
            '0x1212121212121212121212121212121212121212121212121212121212121212',
        },
      ])
    })

    it('compute slots for a simple (partial) struct', () => {
      expect(
        computeStorageSlots(layout, {
          _struct: {
            _structAddress: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
          },
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000013',
          val:
            '0x0000000000000000000000005a0b54d5dc17e0aadc383d2db43b0a0d3e029c4c',
        },
      ])
    })

    it('compute slots for packed variables', () => {
      expect(
        computeStorageSlots(layout, {
          _packedAddress: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
          _packedBool: true,
          _packedBytes11: '0x1212121212121212121212',
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000015',
          val:
            '0x1212121212121212121212015a0b54d5dc17e0aadc383d2db43b0a0d3e029c4c',
        },
      ])
    })

    it('compute slots for packed variables (2)', () => {
      expect(
        computeStorageSlots(layout, {
          _otherPackedBytes11: '0x1212121212121212121212',
          _otherPackedBool: true,
          _otherPackedAddress: '0x5A0b54D5dc17e0AadC383d2db43B0a0D3E029c4c',
        })
      ).to.deep.equal([
        {
          key:
            '0x0000000000000000000000000000000000000000000000000000000000000016',
          val:
            '0x5a0b54d5dc17e0aadc383d2db43b0a0d3e029c4c011212121212121212121212',
        },
      ])
    })
  })
})
