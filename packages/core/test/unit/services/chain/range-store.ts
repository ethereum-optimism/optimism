import '../../../setup'

/* External Imports */
import BigNum from 'bn.js'

/* Internal Imports */
import { RangeStore, BlockRange } from '../../../../src/utils/range-store'

describe('RangeStore', () => {
  let store = new RangeStore()
  beforeEach(() => {
    store = new RangeStore()
  })

  describe('addRange', () => {
    it('should correctly insert a range with no overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }

      const expected = [range1]

      store.addRange(range1)

      store.ranges.should.deep.equal(expected)
    })

    it('should correctly insert a range with full overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }
      const range2 = {
        start: range1.start,
        end: range1.end,
        block: range1.block.addn(1),
      }

      const expected = [range2]

      store.addRange(range1)
      store.addRange(range2)

      store.ranges.should.deep.equal(expected)
    })

    it('should correctly insert a range with partial left overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }
      const range2 = {
        start: range1.start.subn(50),
        end: range1.end.subn(50),
        block: range1.block.addn(1),
      }

      const expected = [
        range2,
        {
          ...range1,
          ...{ start: range2.end },
        },
      ]

      store.addRange(range1)
      store.addRange(range2)

      store.ranges.should.deep.equal(expected)
    })

    it('should correctly insert a range with partial right overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }
      const range2 = {
        start: range1.start.addn(50),
        end: range1.end.addn(50),
        block: range1.block.addn(1),
      }

      const expected = [
        {
          ...range1,
          ...{ end: range2.start },
        },
        range2,
      ]

      store.addRange(range1)
      store.addRange(range2)

      store.ranges.should.deep.equal(expected)
    })

    it('should correctly insert a range with multiple overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }
      const range2 = {
        start: new BigNum(300),
        end: new BigNum(400),
        block: new BigNum(0),
      }
      const range3 = {
        start: range1.start,
        end: range2.end,
        block: new BigNum(1),
      }

      const expected = [range3]

      store.addRange(range1)
      store.addRange(range2)
      store.addRange(range3)

      store.ranges.should.deep.equal(expected)
    })

    it('should not insert a range with a lower block number', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(1),
      }
      const range2 = {
        start: range1.start,
        end: range1.end,
        block: range1.block.subn(1),
      }

      const expected = [range1]

      store.addRange(range1)
      store.addRange(range2)

      store.ranges.should.deep.equal(expected)
    })
  })

  describe('removeRange', () => {
    it('should correctly remove a range with full overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }
      const remove1 = {
        start: range1.start,
        end: range1.end,
      }

      const expected: BlockRange[] = []

      store.addRange(range1)
      store.removeRange(remove1)

      store.ranges.should.deep.equal(expected)
    })

    it('should correctly remove a range with partial left overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }
      const remove1 = {
        start: range1.start.subn(50),
        end: range1.end.subn(50),
      }

      const expected = [
        {
          ...range1,
          ...{
            start: remove1.end,
          },
        },
      ]

      store.addRange(range1)
      store.removeRange(remove1)

      store.ranges.should.deep.equal(expected)
    })

    it('should correctly remove a range with partial right overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }
      const remove1 = {
        start: range1.start.addn(50),
        end: range1.end.addn(50),
      }

      const expected = [
        {
          ...range1,
          ...{
            end: remove1.start,
          },
        },
      ]

      store.addRange(range1)
      store.removeRange(remove1)

      store.ranges.should.deep.equal(expected)
    })

    it('should correctly remove a range with multiple overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }
      const range2 = {
        start: new BigNum(300),
        end: new BigNum(400),
        block: new BigNum(0),
      }
      const remove1 = {
        start: range1.start,
        end: range2.end,
      }

      const expected: BlockRange[] = []

      store.addRange(range1)
      store.removeRange(remove1)

      store.ranges.should.deep.equal(expected)
    })

    it('should not remove a range with no overlap', () => {
      const range1 = {
        start: new BigNum(100),
        end: new BigNum(200),
        block: new BigNum(0),
      }
      const remove1 = {
        start: new BigNum(0),
        end: new BigNum(100),
      }

      const expected = [range1]

      store.addRange(range1)
      store.removeRange(remove1)

      store.ranges.should.deep.equal(expected)
    })
  })
})
