import '../../setup'
import {
  BigNumber,
  ZERO,
  ONE,
  LITTLE_ENDIAN,
  BIG_ENDIAN,
  TWO,
} from '../../../src/app/utils'
import * as assert from 'assert'

describe('BigNumber', () => {
  describe('constructor', () => {
    it('should handle number', () => {
      const num: BigNumber = new BigNumber(10)
      assert(num.toNumber() === 10, 'Could not construct BigNumber with number')
    })

    it('should handle base10 string', () => {
      const num: BigNumber = new BigNumber('10', 10)
      assert(num.toNumber() === 10, 'Could not construct BigNumber with number')
    })

    it('should handle hex string', () => {
      const num: BigNumber = new BigNumber('a', 'hex')
      assert(num.toNumber() === 10, 'Could not construct BigNumber with number')
    })

    it('should handle BigNumber', () => {
      const num: BigNumber = new BigNumber(new BigNumber(10))
      assert(num.toNumber() === 10, 'Could not construct BigNumber with number')
    })

    it('should handle Buffer', () => {
      const num: BigNumber = new BigNumber(new BigNumber(10).toBuffer())
      assert(num.toNumber() === 10, 'Could not construct BigNumber with number')
    })
  })

  describe('min', () => {
    it('first is less', () => {
      assert(BigNumber.min(ZERO, ONE).eq(ZERO))
    })

    it('second is less', () => {
      assert(BigNumber.min(ONE, ZERO).eq(ZERO))
    })

    it('they are equal', () => {
      assert(BigNumber.min(ZERO, ZERO).eq(ZERO))
    })
  })

  describe('max', () => {
    it('first is greater', () => {
      assert(BigNumber.max(ONE, ZERO).eq(ONE))
    })

    it('second is greater', () => {
      assert(BigNumber.max(ZERO, ONE).eq(ONE))
    })

    it('they are equal', () => {
      assert(BigNumber.max(ZERO, ZERO).eq(ZERO))
    })
  })

  describe('isBigNumber', () => {
    it('test BigNum', () => {
      assert(BigNumber.isBigNumber(ONE))
    })

    it('test number', () => {
      assert(!BigNumber.isBigNumber(1))
    })

    it('test undefined', () => {
      assert(!BigNumber.isBigNumber(undefined))
    })

    it('test null', () => {
      assert(!BigNumber.isBigNumber(null))
    })
  })

  describe('clone', () => {
    it('test different memory address', () => {
      const clone: BigNumber = ONE.clone()
      assert(clone.eq(ONE) && clone !== ONE)
    })
  })

  describe('toString', () => {
    it('test base 10', () => {
      assert(new BigNumber(11).toString(10) === '11')
      assert(new BigNumber(-11).toString(10) === '-11')
    })

    it('test hex', () => {
      assert(new BigNumber(11).toString(16) === 'b')
      assert(new BigNumber(11).toString('hex') === 'b')
      assert(new BigNumber(-11).toString(16) === '-b')
      assert(new BigNumber(-11).toString('hex') === '-b')
    })
  })

  describe('toJSON', () => {
    it('test positive and negative', () => {
      assert(new BigNumber(11).toJSON() === 'b')
      assert(new BigNumber(-11).toJSON() === '-b')
    })
  })

  describe('toNumber', () => {
    it('test outputs correct number', () => {
      assert(new BigNumber(27).toNumber() === 27)
    })
  })

  describe('add', () => {
    it('test add positive', () => {
      assert(ONE.add(ONE).eq(new BigNumber(2)))
    })

    it('test add negative', () => {
      assert(ONE.add(new BigNumber(-1)).eq(ZERO))
    })
  })

  describe('sub', () => {
    it('test subtract positive', () => {
      assert(ONE.sub(ONE).eq(ZERO))
    })

    it('test subtract negative', () => {
      assert(ZERO.sub(new BigNumber(-1)).eq(ONE))
    })
  })

  describe('mul', () => {
    it('test multiply positive', () => {
      assert(new BigNumber(10).mul(new BigNumber(5)).eq(new BigNumber(50)))
    })

    it('test multiply negative', () => {
      assert(new BigNumber(10).mul(new BigNumber(-5)).eq(new BigNumber(-50)))
    })
  })

  describe('div', () => {
    it('test divide positive', () => {
      assert(new BigNumber(10).div(new BigNumber(5)).eq(new BigNumber(2)))
    })

    it('test divide negative', () => {
      assert(new BigNumber(10).div(new BigNumber(-5)).eq(new BigNumber(-2)))
    })

    it('test divide by 0 throws', () => {
      try {
        ONE.div(ZERO)
        assert(false, 'Divide by negative should have thrown')
      } catch (e) {
        assert(true, 'This should happen')
      }
    })
  })

  describe('divRound', () => {
    it('test divide & round down positive', () => {
      assert(new BigNumber(10).divRound(new BigNumber(3)).eq(new BigNumber(3)))
    })

    it('test divide & round down negative', () => {
      assert(
        new BigNumber(10).divRound(new BigNumber(-3)).eq(new BigNumber(-3))
      )
    })

    it('test divide & round up positive', () => {
      assert(new BigNumber(10).divRound(new BigNumber(9)).eq(new BigNumber(1)))
    })

    it('test divide & round up negative', () => {
      assert(
        new BigNumber(10).divRound(new BigNumber(-9)).eq(new BigNumber(-1))
      )
    })

    it('test divide by 0 throws', () => {
      try {
        ONE.divRound(ZERO)
        assert(false, 'Divide by negative should have thrown')
      } catch (e) {
        assert(true, 'This should happen')
      }
    })
  })

  describe('pow', () => {
    it('test positive power', () => {
      assert(new BigNumber(10).pow(new BigNumber(3)).eq(new BigNumber(1000)))
    })

    it('test negative power, positive result', () => {
      assert(new BigNumber(-10).pow(new BigNumber(3)).eq(new BigNumber(-1000)))
    })

    it('test negative power, negative result', () => {
      assert(new BigNumber(-10).pow(new BigNumber(2)).eq(new BigNumber(100)))
    })

    // None of these work because bn.js does not support them
    // it('test positive fractional power', () => {
    //   console.log(`100^-2 = ${new BigNumber(100).pow(new BigNumber(.5)).toString()}`)
    //   assert(new BigNumber(100).pow(new BigNumber(.5)).eq(new BigNumber(10)))
    // })
    //
    // it('test negative power', () => {
    //   assert(new BigNumber(100).pow(new BigNumber(-2)).eq(new BigNumber(0.0001)))
    // })
    //
    // it('test negative fractional power', () => {
    //   assert(new BigNumber(100).pow(new BigNumber(-.5)).eq(new BigNumber(0.1)))
    // })
  })

  describe('mod', () => {
    it('test positive mod', () => {
      assert(new BigNumber(10).mod(new BigNumber(3)).eq(ONE))
    })
  })

  describe('modNum', () => {
    it('test positive mod', () => {
      assert(new BigNumber(10).modNum(3).eq(ONE))
    })
  })

  describe('abs', () => {
    it('test abs positive', () => {
      assert(new BigNumber(10).abs().eq(new BigNumber(10)))
    })

    it('test abs negative', () => {
      assert(new BigNumber(-10).abs().eq(new BigNumber(10)))
    })
  })

  describe('xor', () => {
    it('works for 0', () => {
      assert(ZERO.xor(ONE).equals(ONE))
      assert(ZERO.xor(ZERO).equals(ZERO))
    })

    it('works for 1', () => {
      assert(ONE.xor(ONE).equals(ZERO))
      assert(ONE.xor(ZERO).equals(ONE))
    })

    it('works for 2', () => {
      assert(TWO.xor(ZERO).equals(TWO))
      assert(TWO.xor(ONE).equals(new BigNumber(3)))
      assert(TWO.xor(TWO).equals(ZERO))
    })
  })

  describe('and', () => {
    it('works for 0', () => {
      assert(ZERO.and(ONE).equals(ZERO))
      assert(ZERO.and(ZERO).equals(ZERO))
    })

    it('works for 1', () => {
      assert(ONE.and(ZERO).equals(ZERO))
      assert(ONE.and(ONE).equals(ONE))
      assert(ONE.and(TWO).equals(ZERO))
    })

    it('works for 2', () => {
      assert(TWO.and(ZERO).equals(ZERO))
      assert(TWO.and(ONE).equals(ZERO))
      assert(TWO.and(TWO).equals(TWO))
    })
  })

  describe('shiftLeft', () => {
    it('works for 0', () => {
      assert(ZERO.shiftLeft(0).equals(ZERO))
      assert(ZERO.shiftLeft(1).equals(ZERO))
      assert(ZERO.shiftLeft(5).equals(ZERO))
    })

    it('works for 1', () => {
      assert(ONE.shiftLeft(0).equals(ONE))
      assert(ONE.shiftLeft(1).equals(TWO))
      assert(ONE.shiftLeft(2).equals(new BigNumber(4)))
    })

    it('works for 2', () => {
      assert(TWO.shiftLeft(0).equals(TWO))
      assert(TWO.shiftLeft(1).equals(new BigNumber(4)))
    })
  })

  describe('shiftRight', () => {
    it('works for 0', () => {
      assert(ZERO.shiftRight(0).equals(ZERO))
      assert(ZERO.shiftRight(1).equals(ZERO))
      assert(ZERO.shiftRight(5).equals(ZERO))
    })

    it('works for 1', () => {
      assert(ONE.shiftRight(0).equals(ONE))
      assert(ONE.shiftRight(1).equals(ZERO))
      assert(ONE.shiftRight(2).equals(ZERO))
    })

    it('works for 2', () => {
      assert(TWO.shiftRight(0).equals(TWO))
      assert(TWO.shiftRight(1).equals(ONE))
      assert(TWO.shiftRight(2).equals(ZERO))
    })
  })

  describe('gt', () => {
    it('test positive', () => {
      assert(new BigNumber(10).gt(new BigNumber(9)))
      assert(!new BigNumber(9).gt(new BigNumber(10)))
      assert(!new BigNumber(10).gt(new BigNumber(10)))
    })

    it('test negative', () => {
      assert(new BigNumber(10).gt(new BigNumber(-11)))
      assert(!new BigNumber(-11).gt(new BigNumber(10)))
    })

    it('test negatives', () => {
      assert(new BigNumber(-10).gt(new BigNumber(-11)))
      assert(!new BigNumber(-11).gt(new BigNumber(-10)))
      assert(!new BigNumber(-10).gt(new BigNumber(-10)))
    })
  })

  describe('gte', () => {
    it('test positive', () => {
      assert(new BigNumber(10).gte(new BigNumber(9)))
      assert(!new BigNumber(9).gte(new BigNumber(10)))
      assert(new BigNumber(10).gte(new BigNumber(10)))
    })

    it('test negative', () => {
      assert(new BigNumber(10).gte(new BigNumber(-11)))
      assert(!new BigNumber(-11).gte(new BigNumber(10)))
    })

    it('test negatives', () => {
      assert(new BigNumber(-10).gte(new BigNumber(-11)))
      assert(!new BigNumber(-11).gte(new BigNumber(-10)))
      assert(new BigNumber(-10).gte(new BigNumber(-10)))
    })
  })

  describe('lt', () => {
    it('test positive', () => {
      assert(!new BigNumber(10).lt(new BigNumber(9)))
      assert(new BigNumber(9).lt(new BigNumber(10)))
      assert(!new BigNumber(10).lt(new BigNumber(10)))
    })

    it('test negative', () => {
      assert(!new BigNumber(10).lt(new BigNumber(-11)))
      assert(new BigNumber(-11).lt(new BigNumber(10)))
    })

    it('test negatives', () => {
      assert(!new BigNumber(-10).lt(new BigNumber(-11)))
      assert(new BigNumber(-11).lt(new BigNumber(-10)))
      assert(!new BigNumber(-10).lt(new BigNumber(-10)))
    })
  })

  describe('lte', () => {
    it('test positive', () => {
      assert(!new BigNumber(10).lte(new BigNumber(9)))
      assert(new BigNumber(9).lte(new BigNumber(10)))
      assert(new BigNumber(10).lte(new BigNumber(10)))
    })

    it('test negative', () => {
      assert(!new BigNumber(10).lte(new BigNumber(-11)))
      assert(new BigNumber(-11).lte(new BigNumber(10)))
    })

    it('test negatives', () => {
      assert(!new BigNumber(-10).lte(new BigNumber(-11)))
      assert(new BigNumber(-11).lte(new BigNumber(-10)))
      assert(new BigNumber(-10).lte(new BigNumber(-10)))
    })
  })

  describe('eq', () => {
    it('test positive', () => {
      assert(new BigNumber(10).eq(new BigNumber(10)))
      assert(!new BigNumber(9).eq(new BigNumber(10)))
    })

    it('test negative', () => {
      assert(new BigNumber(-10).eq(new BigNumber(-10)))
      assert(!new BigNumber(-11).eq(new BigNumber(-10)))
    })
  })

  describe('compare', () => {
    it('test first is less', () => {
      assert(new BigNumber(10).compare(new BigNumber(11)) === -1)
      assert(new BigNumber(-11).compare(new BigNumber(-10)) === -1)
    })

    it('test second is less', () => {
      assert(new BigNumber(11).compare(new BigNumber(10)) === 1)
      assert(new BigNumber(-10).compare(new BigNumber(-11)) === 1)
    })

    it('test equal', () => {
      assert(new BigNumber(10).compare(new BigNumber(10)) === 0)
      assert(new BigNumber(-10).compare(new BigNumber(-10)) === 0)
    })
  })
})
