import '../../setup'

/* External Imports */
import { areEqual, objectsEqual } from '@eth-optimism/core-utils'
import * as assert from 'assert'

/* Internal Imports */
import {
  EqualityDecider,
  EqualityDeciderInput,
} from '../../../src/app/deciders'
import { Decision } from '../../../src/types'

describe('EqualityDecider', () => {
  const decider: EqualityDecider = EqualityDecider.instance()

  describe('decide', () => {
    describe('true', () => {
      describe('no values', () => {
        it('should return true with empty input', async () => {
          const decision: Decision = await decider.decide({
            itemsToCompare: [],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)
        })
      })

      describe('single value', () => {
        it('should return true with single primitive input', async () => {
          const decision: Decision = await decider.decide({
            itemsToCompare: [1],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          input.itemsToCompare[0].should.eq(1)
        })

        it('should return true with single object input', async () => {
          const obj: {} = { test: 'Wooooo!' }
          const decision: Decision = await decider.decide({
            itemsToCompare: [obj],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(objectsEqual(input.itemsToCompare[0], obj))
        })

        it('should return true with single array input', async () => {
          const arr: number[] = [1]
          const decision: Decision = await decider.decide({
            itemsToCompare: [arr],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(areEqual(input.itemsToCompare[0], arr))
        })
      })

      describe('two values', () => {
        it('should return true with two primitive inputs', async () => {
          const decision: Decision = await decider.decide({
            itemsToCompare: [1, 1],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          input.itemsToCompare[0].should.eq(1)
          input.itemsToCompare[1].should.eq(1)
        })

        it('should return true with two object inputs', async () => {
          const obj: {} = { test: 'Wooooo!' }
          const decision: Decision = await decider.decide({
            itemsToCompare: [obj, { test: 'Wooooo!' }],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(objectsEqual(input.itemsToCompare[0], obj))
          assert(objectsEqual(input.itemsToCompare[1], obj))
        })

        it('should return true with two array inputs', async () => {
          const arr: number[] = [1]
          const decision: Decision = await decider.decide({
            itemsToCompare: [arr, [1]],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(areEqual(input.itemsToCompare[0], arr))
          assert(areEqual(input.itemsToCompare[1], arr))
        })
      })

      describe('three values', () => {
        it('should return true with three primitive inputs', async () => {
          const decision: Decision = await decider.decide({
            itemsToCompare: [1, 1, 1],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          input.itemsToCompare[0].should.eq(1)
          input.itemsToCompare[1].should.eq(1)
          input.itemsToCompare[2].should.eq(1)
        })

        it('should return true with three object inputs', async () => {
          const obj: {} = { test: 'Wooooo!' }
          const decision: Decision = await decider.decide({
            itemsToCompare: [obj, { test: 'Wooooo!' }, obj],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(objectsEqual(input.itemsToCompare[0], obj))
          assert(objectsEqual(input.itemsToCompare[1], obj))
          assert(objectsEqual(input.itemsToCompare[2], obj))
        })

        it('should return true with three array inputs', async () => {
          const arr: number[] = [1]
          const decision: Decision = await decider.decide({
            itemsToCompare: [arr, [1], arr],
          })

          decision.outcome.should.eq(true)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(areEqual(input.itemsToCompare[0], arr))
          assert(areEqual(input.itemsToCompare[1], arr))
          assert(areEqual(input.itemsToCompare[2], arr))
        })
      })
    })

    describe('false', () => {
      describe('two values', () => {
        it('should return false with two primitive inputs', async () => {
          const decision: Decision = await decider.decide({
            itemsToCompare: [1, 2],
          })

          decision.outcome.should.eq(false)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          input.itemsToCompare[0].should.eq(1)
          input.itemsToCompare[1].should.eq(2)

          assert(
            areEqual(
              input.itemsToCompare,
              decision.justification[0].implicationWitness
            )
          )
        })

        it('should return false with two object inputs', async () => {
          const obj: {} = { test: 'Wooooo!' }
          const diff: {} = { test: 'Woo!' }
          const decision: Decision = await decider.decide({
            itemsToCompare: [obj, diff],
          })

          decision.outcome.should.eq(false)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(objectsEqual(input.itemsToCompare[0], obj))
          assert(objectsEqual(input.itemsToCompare[1], diff))

          assert(
            areEqual(
              input.itemsToCompare,
              decision.justification[0].implicationWitness
            )
          )
        })

        it('should return false with two array inputs', async () => {
          const arr: number[] = [1]
          const diff: number[] = [2]
          const decision: Decision = await decider.decide({
            itemsToCompare: [arr, diff],
          })

          decision.outcome.should.eq(false)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(areEqual(input.itemsToCompare[0], arr))
          assert(areEqual(input.itemsToCompare[1], diff))

          assert(
            areEqual(
              input.itemsToCompare,
              decision.justification[0].implicationWitness
            )
          )
        })
      })

      describe('three values', () => {
        it('should return false with three primitive inputs', async () => {
          const decision: Decision = await decider.decide({
            itemsToCompare: [1, 1, 2],
          })

          decision.outcome.should.eq(false)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          input.itemsToCompare[0].should.eq(1)
          input.itemsToCompare[1].should.eq(1)
          input.itemsToCompare[2].should.eq(2)

          assert(areEqual([1, 2], decision.justification[0].implicationWitness))
        })

        it('should return false with three object inputs', async () => {
          const obj: {} = { test: 'Wooooo!' }
          const diff: {} = { test: 'Woo!' }
          const decision: Decision = await decider.decide({
            itemsToCompare: [obj, obj, diff],
          })

          decision.outcome.should.eq(false)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(objectsEqual(input.itemsToCompare[0], obj))
          assert(objectsEqual(input.itemsToCompare[1], obj))
          assert(objectsEqual(input.itemsToCompare[2], diff))

          assert(
            areEqual([obj, diff], decision.justification[0].implicationWitness)
          )
        })

        it('should return false with three array inputs', async () => {
          const arr: number[] = [1]
          const diff: number[] = [2]
          const decision: Decision = await decider.decide({
            itemsToCompare: [arr, arr, diff],
          })

          decision.outcome.should.eq(false)
          decision.justification.length.should.eq(1)
          decision.justification[0].implication.decider.should.eq(decider)

          const input: EqualityDeciderInput = decision.justification[0]
            .implication.input as EqualityDeciderInput
          assert(areEqual(input.itemsToCompare[0], arr))
          assert(areEqual(input.itemsToCompare[1], arr))
          assert(areEqual(input.itemsToCompare[2], diff))

          assert(
            areEqual([arr, diff], decision.justification[0].implicationWitness)
          )
        })
      })
    })
  })
})
