/* External Imports */
import { bytecodeToBuffer } from '@pigi/rollup-core'
import { bufferUtils } from '@pigi/core-utils'

/* Internal Imports */
import { should } from '../setup'
import {
  EvmErrors,
  EvmIntrospectionUtil,
  ExecutionResult,
  ExecutionResultComparison,
  StepContext,
} from '../../src/types/vm'
import { EvmIntrospectionUtilImpl } from '../../src/tools/vm'
import {
  invalidBytesConsumedBytecode,
  memoryAndStackBytecode,
  returnNumberBytecode,
  voidBytecode,
} from '../helpers'

const empty: Buffer = Buffer.from('', 'hex')

describe('EvmIntrospectionUtil', () => {
  let evmUtil: EvmIntrospectionUtil

  beforeEach(async () => {
    evmUtil = await EvmIntrospectionUtilImpl.create()
  })

  describe('getExecutionResult', () => {
    it('handles empty case', async () => {
      const res: ExecutionResult = await evmUtil.getExecutionResult(empty)

      should.not.exist(
        res.error,
        'Simple bytecode to return a number yielded error!'
      )
      res.result.should.eql(empty, 'Got unexpected result!')
    })

    it('gets execution result of simple bytecode to return a number', async () => {
      const num: number = 1
      const res: ExecutionResult = await evmUtil.getExecutionResult(
        bytecodeToBuffer(returnNumberBytecode(num))
      )

      should.not.exist(
        res.error,
        'Simple bytecode to return a number yielded error!'
      )
      res.result.should.eql(
        bufferUtils.numberToBuffer(num),
        'Got unexpected result!'
      )
    })

    it('handles void case', async () => {
      const res: ExecutionResult = await evmUtil.getExecutionResult(
        bytecodeToBuffer(voidBytecode)
      )

      should.not.exist(
        res.error,
        'Simple bytecode to return a number yielded error!'
      )
      res.result.should.eql(empty, 'Got unexpected result!')
    })

    it('handles errors', async () => {
      const res: ExecutionResult = await evmUtil.getExecutionResult(
        bytecodeToBuffer(invalidBytesConsumedBytecode)
      )

      should.exist(res.error, 'Invalid bytecode should yield error!')
      res.error.should.equal(EvmErrors.STACK_UNDERFLOW_ERROR)

      res.result.should.eql(empty, 'Got unexpected result!')
    })
  })

  describe('getExecutionResultComparison', () => {
    it('handles empty case', async () => {
      const res: ExecutionResultComparison = await evmUtil.getExecutionResultComparison(
        empty,
        empty
      )

      res.resultsDiffer.should.equal(false, 'Results differ mismatch!')
      res.firstResult.result.should.eql(
        res.secondResult.result,
        'Result mismatch!'
      )
    })

    it('handles different bytecode with same output case', async () => {
      const res: ExecutionResultComparison = await evmUtil.getExecutionResultComparison(
        empty,
        bytecodeToBuffer(voidBytecode)
      )

      res.resultsDiffer.should.equal(false, 'Results differ mismatch!')
      res.firstResult.result.should.eql(
        res.secondResult.result,
        'Result mismatch!'
      )
    })

    it('ensures results differ when they should', async () => {
      const res: ExecutionResultComparison = await evmUtil.getExecutionResultComparison(
        bytecodeToBuffer(returnNumberBytecode(1)),
        bytecodeToBuffer(voidBytecode)
      )

      res.resultsDiffer.should.equal(true, 'Results differ mismatch!')
      res.firstResult.result.should.eql(
        bufferUtils.numberToBuffer(1),
        'first result mismatch!'
      )
      res.secondResult.result.should.eql(empty, 'Result mismatch!')
    })

    it('ensures non-void results match', async () => {
      const res: ExecutionResultComparison = await evmUtil.getExecutionResultComparison(
        bytecodeToBuffer(returnNumberBytecode(1)),
        bytecodeToBuffer(returnNumberBytecode(1))
      )

      res.resultsDiffer.should.equal(false, 'Results differ mismatch!')
      res.firstResult.should.eql(res.secondResult, 'Result mismatch!')
    })

    it('ensures results match on error', async () => {
      const res: ExecutionResultComparison = await evmUtil.getExecutionResultComparison(
        bytecodeToBuffer(invalidBytesConsumedBytecode),
        bytecodeToBuffer(invalidBytesConsumedBytecode)
      )

      res.resultsDiffer.should.equal(false, 'Results differ mismatch!')
      res.firstResult.should.eql(res.secondResult, 'Result mismatch!')
    })
  })

  describe('getStepContextBeforeStep', () => {
    it('handles empty case', async () => {
      const ctx: StepContext = await evmUtil.getStepContextBeforeStep(empty, 1)

      should.not.exist(
        ctx,
        'Context should not exist before empty bytecode execution!'
      )
    })

    it('is undefined if step is not hit', async () => {
      const ctx: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(memoryAndStackBytecode),
        3
      )

      should.not.exist(
        ctx,
        'Context should not exist since PC index is not hit!'
      )
    })

    it('works for empty memory & stack', async () => {
      const ctx: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(voidBytecode),
        0
      )

      should.exist(ctx, 'Empty memory and stack context should exist!')

      ctx.pc.should.equal(0, 'PC mismatch!')
      ctx.opcode.should.equal(voidBytecode[0].opcode, 'Opcode mismatch!')
      ctx.stackDepth.should.equal(0, 'Stack depth mismatch!')
      ctx.stack.should.eql([], 'Stack mismatch!')
      ctx.memoryWordCount.should.equal(0, 'Memory word count mismatch!')
      ctx.memory.should.eql(empty, 'Memory mismatch!')
    })

    it('works for populated memory & stack', async () => {
      const ctx: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(memoryAndStackBytecode),
        38
      )

      should.exist(ctx, 'Memory and stack context should exist!')

      ctx.pc.should.equal(38, 'PC mismatch!')
      ctx.opcode.should.equal(
        memoryAndStackBytecode[4].opcode,
        'Opcode mismatch!'
      )
      ctx.stackDepth.should.equal(1, 'Stack depth mismatch!')
      ctx.stack[0].should.eql(
        memoryAndStackBytecode[0].consumedBytes,
        'Stack mismatch!'
      )
      ctx.memoryWordCount.should.equal(4, 'Memory word count mismatch!')
      ctx.memory.should.eql(
        Buffer.from('00'.repeat(127) + '01', 'hex'),
        'Memory mismatch!'
      )
    })

    it('handles case where code errors after step', async () => {
      const ctx: StepContext = await evmUtil.getStepContextBeforeStep(
        bytecodeToBuffer(invalidBytesConsumedBytecode),
        2
      )

      should.exist(ctx, 'Context should exist!')

      ctx.pc.should.equal(2, 'PC mismatch!')
      ctx.opcode.should.equal(
        invalidBytesConsumedBytecode[1].opcode,
        'Opcode mismatch!'
      )
      ctx.stackDepth.should.equal(1, 'Stack depth mismatch!')
      ctx.stack[0].should.eql(
        invalidBytesConsumedBytecode[0].consumedBytes,
        'Stack mismatch!'
      )
      ctx.memoryWordCount.should.equal(0, 'Memory word count mismatch!')
      ctx.memory.should.eql(empty, 'Memory mismatch!')
    })
  })
})
