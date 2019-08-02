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
        left: {
          decider: trueDecider,
          input: leftInput,
        },
        leftWitness,
        right: {
          decider: trueDecider,
          input: rightInput,
        },
        rightWitness,
      }

      const decision: Decision = await decider.decide(andInput)

      decision.outcome.should.eq(true)
      decision.justification.length.should.eq(3)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(trueDecider)
      decision.justification[2].implication.decider.should.eq(trueDecider)

      decision.justification[1].implication.input.should.eq(leftInput)
      decision.justification[2].implication.input.should.eq(rightInput)

      decision.justification[1].implicationWitness.should.eq(leftWitness)
      decision.justification[2].implicationWitness.should.eq(rightWitness)
    })

    it('should return false with left false', async () => {
      const andInput: AndDeciderInput = {
        left: {
          decider: falseDecider,
          input: leftInput,
        },
        leftWitness,
        right: {
          decider: trueDecider,
          input: rightInput,
        },
        rightWitness,
      }

      const decision: Decision = await decider.decide(andInput)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)

      decision.justification[1].implication.input.should.eq(leftInput)
      decision.justification[1].implicationWitness.should.eq(leftWitness)
    })

    it('should return false with right false', async () => {
      const andInput: AndDeciderInput = {
        left: {
          decider: trueDecider,
          input: leftInput,
        },
        leftWitness,
        right: {
          decider: falseDecider,
          input: rightInput,
        },
        rightWitness,
      }

      const decision: Decision = await decider.decide(andInput, undefined)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)

      decision.justification[1].implication.input.should.eq(rightInput)
      decision.justification[1].implicationWitness.should.eq(rightWitness)
    })

    it('should return false with both false', async () => {
      const andInput: AndDeciderInput = {
        left: {
          decider: falseDecider,
          input: leftInput,
        },
        leftWitness,
        right: {
          decider: falseDecider,
          input: rightInput,
        },
        rightWitness,
      }

      const decision: Decision = await decider.decide(andInput)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)

      decision.justification[1].implication.input.should.eq(leftInput)
      decision.justification[1].implicationWitness.should.eq(leftWitness)
    })

    it('should throw when left cannot decide', async () => {
      const andInput: AndDeciderInput = {
        left: {
          decider: cannotDecideDecider,
          input: leftInput,
        },
        leftWitness,
        right: {
          decider: trueDecider,
          input: rightInput,
        },
        rightWitness,
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
        left: {
          decider: trueDecider,
          input: leftInput,
        },
        leftWitness,
        right: {
          decider: cannotDecideDecider,
          input: rightInput,
        },
        rightWitness,
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
        left: {
          decider: cannotDecideDecider,
          input: leftInput,
        },
        leftWitness,
        right: {
          decider: cannotDecideDecider,
          input: rightInput,
        },
        rightWitness,
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
