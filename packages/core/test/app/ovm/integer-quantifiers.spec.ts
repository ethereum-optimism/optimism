/* Internal Imports */
import {
  IntegerRangeQuantifier,
  NonnegativeIntegerLessThanQuantifier,
} from '../../../src/app'

describe('IntegerQuantifiers', () => {
  describe('IntegerRangeQuantifier', () => {
    it('should quantify a positive range', async () => {
      const quantifier = new IntegerRangeQuantifier()
      const range = quantifier.getAllQuantified({ start: 100, end: 105 })
      range.should.deep.equal({
        results: [100, 101, 102, 103, 104],
        allResultsQuantified: true,
      })
    })

    it('should quantify a large positive range', async () => {
      const quantifier = new IntegerRangeQuantifier()
      const range = quantifier.getAllQuantified({ start: 100, end: 1005 })
      // Generate a range from 100 to 1005
      const expectedResult = []
      for (let i = 100; i < 1005; i++) {
        expectedResult.push(i)
      }
      range.should.deep.equal({
        results: expectedResult,
        allResultsQuantified: true,
      })
    })

    it('should quantify a negative range', async () => {
      const quantifier = new IntegerRangeQuantifier()
      const range = quantifier.getAllQuantified({ start: -105, end: -100 })
      range.should.deep.equal({
        results: [-105, -104, -103, -102, -101],
        allResultsQuantified: true,
      })
    })

    it('should quantify a range with a negative start & positive end', async () => {
      const quantifier = new IntegerRangeQuantifier()
      const range = quantifier.getAllQuantified({ start: -3, end: 2 })
      range.should.deep.equal({
        results: [-3, -2, -1, 0, 1],
        allResultsQuantified: true,
      })
    })

    it('should throw an error if end < start ', async () => {
      const quantifier = new IntegerRangeQuantifier()
      const callGetQuantified = () =>
        quantifier.getAllQuantified({ start: 100, end: 95 })
      callGetQuantified.should.throw()
    })

    it('should return an empty array if start == end', async () => {
      const quantifier = new IntegerRangeQuantifier()
      const range = quantifier.getAllQuantified({ start: 100, end: 100 })
      range.should.deep.equal({
        results: [],
        allResultsQuantified: true,
      })
    })
  })
  describe('NonnegativeIntegerLessThanQuantifier', () => {
    it('should quantify numbers less than 5', async () => {
      const quantifier = new NonnegativeIntegerLessThanQuantifier()
      const range = quantifier.getAllQuantified(5)
      range.should.deep.equal({
        results: [0, 1, 2, 3, 4],
        allResultsQuantified: true,
      })
    })

    it('should throw an error if attempting to quantify nonnegative numbers less than 0', async () => {
      const quantifier = new NonnegativeIntegerLessThanQuantifier()
      const callGetQuantified = () => quantifier.getAllQuantified(-5)
      callGetQuantified.should.throw()
    })

    it('should return an empty array if quantifying `less than 0`', async () => {
      const quantifier = new NonnegativeIntegerLessThanQuantifier()
      const range = quantifier.getAllQuantified(0)
      range.should.deep.equal({
        results: [],
        allResultsQuantified: true,
      })
    })
  })
})
