import '../../../setup'

import {
  AndDecider,
  AndDeciderInput,
  CannotDecideError,
} from '../../../../src/app/ovm/deciders'
import { CannotDecideDecider, FalseDecider, TrueDecider } from './utils'
import { Decision } from '../../../../src/types/ovm'
import * as assert from 'assert'

describe('AndDecider', () => {
  let decider: AndDecider
  const leftInput: string = 'test'
  const leftWitness: string = 'witness'
  const rightInput: string = 'test 2'
  const rightWitness: string = 'witness 2'

  const trueDecider: TrueDecider = new TrueDecider()
  const falseDecider: FalseDecider = new FalseDecider()
  const cannotDecideDecider: CannotDecideDecider = new CannotDecideDecider()
  beforeEach(() => {
    decider = new AndDecider()
  })

  describe('decide', () => {
    it('should return true with two true decisions', async () => {
      const andInput: AndDeciderInput = {
        properties: [
          {
            decider: trueDecider,
            input: leftInput,
          },
          {
            decider: trueDecider,
            input: rightInput,
          },
        ],
      }

      const decision: Decision = await decider.decide(andInput)

      decision.outcome.should.eq(true)
      decision.justification.length.should.eq(3)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(trueDecider)
      decision.justification[2].implication.decider.should.eq(trueDecider)

      decision.justification[1].implication.input.should.eq(leftInput)
      decision.justification[2].implication.input.should.eq(rightInput)
    })

    it('should return false with left false', async () => {
      const andInput: AndDeciderInput = {
        properties: [
          {
            decider: falseDecider,
            input: leftInput,
          },
          {
            decider: trueDecider,
            input: rightInput,
          },
        ],
      }

      const decision: Decision = await decider.decide(andInput)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)

      decision.justification[1].implication.input.should.eq(leftInput)
    })

    it('should return false with right false', async () => {
      const andInput: AndDeciderInput = {
        properties: [
          {
            decider: trueDecider,
            input: leftInput,
          },
          {
            decider: falseDecider,
            input: rightInput,
          },
        ],
      }

      const decision: Decision = await decider.decide(andInput, undefined)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)

      decision.justification[1].implication.input.should.eq(rightInput)
    })

    it('should return false with both false', async () => {
      const andInput: AndDeciderInput = {
        properties: [
          {
            decider: falseDecider,
            input: leftInput,
          },
          {
            decider: falseDecider,
            input: rightInput,
          },
        ],
      }

      const decision: Decision = await decider.decide(andInput)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)

      decision.justification[1].implication.input.should.eq(leftInput)
    })

    it('should throw when left cannot decide', async () => {
      const andInput: AndDeciderInput = {
        properties: [
          {
            decider: cannotDecideDecider,
            input: leftInput,
          },
          {
            decider: trueDecider,
            input: rightInput,
          },
        ],
      }

      try {
        await decider.decide(andInput, undefined)
        assert(false, 'This should throw a CannotDecideError')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(false, 'Error thrown should be CannotDecideError')
        }
      }
    })

    it('should throw when right cannot decide', async () => {
      const andInput: AndDeciderInput = {
        properties: [
          {
            decider: trueDecider,
            input: leftInput,
          },
          {
            decider: cannotDecideDecider,
            input: rightInput,
          },
        ],
      }

      try {
        await decider.decide(andInput)
        assert(false, 'This should throw a CannotDecideError')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(false, 'Error thrown should be CannotDecideError')
        }
      }
    })

    it('should throw when both cannot decide', async () => {
      const andInput: AndDeciderInput = {
        properties: [
          {
            decider: cannotDecideDecider,
            input: leftInput,
          },
          {
            decider: cannotDecideDecider,
            input: rightInput,
          },
        ],
      }

      try {
        await decider.decide(andInput)
        assert(false, 'This should throw a CannotDecideError')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(false, 'Error thrown should be CannotDecideError')
        }
      }
    })
  })
})
