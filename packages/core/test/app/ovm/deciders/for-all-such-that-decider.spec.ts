import '../../../setup'

import {
  ForAllSuchThatDecider,
  ForAllSuchThatInput,
  CannotDecideError,
} from '../../../../src/app/ovm/deciders'
import { CannotDecideDecider, FalseDecider, TrueDecider } from './utils'
import {
  Decision,
  Property,
  PropertyFactory,
  Quantifier,
  QuantifierResult,
  WitnessFactory,
} from '../../../../src/types/ovm'
import * as assert from 'assert'

/*******************
 * Mocks & Helpers *
 *******************/

class DummyQuantifier implements Quantifier {
  public async getAllQuantified(parameters: any): Promise<QuantifierResult> {
    return undefined
  }
}

const getQuantifierThatReturns = (
  results: any[],
  allResultsQuantified: boolean
): Quantifier => {
  const quantifier: Quantifier = new DummyQuantifier()
  quantifier.getAllQuantified = async (params): Promise<QuantifierResult> => {
    return {
      results,
      allResultsQuantified,
    }
  }
  return quantifier
}

const getPropertyFactoryThatReturns = (
  properties: Property[]
): PropertyFactory => {
  return (input: any): Property => {
    return properties.shift()
  }
}

/*********
 * TESTS *
 *********/

describe('ForAllSuchThatDecider', () => {
  let decider: ForAllSuchThatDecider

  const trueDecider: TrueDecider = new TrueDecider()
  const falseDecider: FalseDecider = new FalseDecider()
  const cannotDecideDecider: CannotDecideDecider = new CannotDecideDecider()
  beforeEach(() => {
    decider = new ForAllSuchThatDecider()
  })

  describe('decide', () => {
    it('should return true with 0 decisions', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([]),
      }

      const decision: Decision = await decider.decide(input)

      decision.outcome.should.eq(true)
      decision.justification.length.should.eq(1)
      decision.justification[0].implication.decider.should.eq(decider)
    })

    it('should work with undefined witness factory', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([]),
      }

      const decision: Decision = await decider.decide(input)

      decision.outcome.should.eq(true)
      decision.justification.length.should.eq(1)
      decision.justification[0].implication.decider.should.eq(decider)
    })

    it('should return true with single true decision', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: trueDecider, input: undefined },
        ]),
      }

      const decision: Decision = await decider.decide(input)

      decision.outcome.should.eq(true)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(trueDecider)
    })

    it('should return true with multiple true decisions', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1, 2, 3], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: trueDecider, input: undefined },
          { decider: trueDecider, input: undefined },
          { decider: trueDecider, input: undefined },
        ]),
      }

      const decision: Decision = await decider.decide(input)

      decision.outcome.should.eq(true)
      decision.justification.length.should.eq(4)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(trueDecider)
      decision.justification[2].implication.decider.should.eq(trueDecider)
      decision.justification[3].implication.decider.should.eq(trueDecider)
    })

    it('should return false with a single false decision', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: falseDecider, input: undefined },
        ]),
      }

      const decision: Decision = await decider.decide(input)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)
    })

    it('should return false with a single false decision in multiple deciders', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1, 2, 3], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: trueDecider, input: undefined },
          { decider: falseDecider, input: undefined },
          { decider: trueDecider, input: undefined },
        ]),
      }

      const decision: Decision = await decider.decide(input)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)
    })

    it('should return false with a single false decision in multiple deciders, some undecided', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1, 2, 3], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: trueDecider, input: undefined },
          { decider: cannotDecideDecider, input: undefined },
          { decider: falseDecider, input: undefined },
        ]),
      }

      const decision: Decision = await decider.decide(input)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)
    })

    it('should throw undecided with single undecided', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: cannotDecideDecider, input: undefined },
        ]),
      }

      try {
        await decider.decide(input)
        assert(false, 'this should have thrown')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(
            false,
            `CannotDecideError expected, but got ${JSON.stringify(e)}`
          )
        }
      }
    })

    it('should throw undecided with single undecided in multiple undecided', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1, 2, 3], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: cannotDecideDecider, input: undefined },
          { decider: cannotDecideDecider, input: undefined },
          { decider: cannotDecideDecider, input: undefined },
        ]),
      }

      try {
        await decider.decide(input)
        assert(false, 'this should have thrown')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(
            false,
            `CannotDecideError expected, but got ${JSON.stringify(e)}`
          )
        }
      }
    })

    it('should throw undecided with single undecided in multiple true', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1, 2, 3], true),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: trueDecider, input: undefined },
          { decider: trueDecider, input: undefined },
          { decider: cannotDecideDecider, input: undefined },
        ]),
      }

      try {
        await decider.decide(input)
        assert(false, 'this should have thrown')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(
            false,
            `CannotDecideError expected, but got ${JSON.stringify(e)}`
          )
        }
      }
    })

    it('should throw undecided with true decisions when not all results quantified', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1, 2, 3], false),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: trueDecider, input: undefined },
          { decider: trueDecider, input: undefined },
          { decider: trueDecider, input: undefined },
        ]),
      }

      try {
        await decider.decide(input)
        assert(false, 'this should have thrown')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(
            false,
            `CannotDecideError expected, but got ${JSON.stringify(e)}`
          )
        }
      }
    })

    it('should throw undecided with true and undecided decisions when not all results quantified', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1, 2, 3], false),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: trueDecider, input: undefined },
          { decider: cannotDecideDecider, input: undefined },
          { decider: trueDecider, input: undefined },
        ]),
      }

      try {
        await decider.decide(input)
        assert(false, 'this should have thrown')
      } catch (e) {
        if (!(e instanceof CannotDecideError)) {
          assert(
            false,
            `CannotDecideError expected, but got ${JSON.stringify(e)}`
          )
        }
      }
    })

    it('should decide false with any false decision when not all results quantified', async () => {
      const input: ForAllSuchThatInput = {
        quantifier: getQuantifierThatReturns([1, 2, 3], false),
        quantifierParameters: undefined,
        propertyFactory: getPropertyFactoryThatReturns([
          { decider: trueDecider, input: undefined },
          { decider: cannotDecideDecider, input: undefined },
          { decider: falseDecider, input: undefined },
        ]),
      }

      const decision: Decision = await decider.decide(input)

      decision.outcome.should.eq(false)
      decision.justification.length.should.eq(2)
      decision.justification[0].implication.decider.should.eq(decider)
      decision.justification[1].implication.decider.should.eq(falseDecider)
    })
  })
})
