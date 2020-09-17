/* External Imports */
import { Contract } from 'ethers'

/* Internal Imports */
import { TestStep } from './test.types'

export interface TestCallGenerator {
  getCalldata(): string
  getReturnData(): string
}

export class DefaultTestGenerator implements TestCallGenerator {
  constructor(
    protected ovmExecutionManager: Contract,
    protected ovmCallHelper: Contract,
    protected step: TestStep
  ) {}

  getFunctionParams(): any[] {
      return this.step.functionParams
  }

  getReturnValues(): any[] {
      return this.step.returnValues
  }

  getCalldata(): string {
      return this.ovmExecutionManager.interface.encodeFunctionData(
          this.step.functionName,
          this.getFunctionParams()
      )
  }

  getReturnData(): string {
      return this.ovmExecutionManager.interface.encodeFunctionResult(
          this.step.functionName,
          this.getReturnValues()
      )
  }
}

export class ovmCALLGenerator extends DefaultTestGenerator {
  getCalleeGenerators(): TestCallGenerator[] {
      return (this.step.functionParams[2] as TestStep[]).map((step) => {
          return getTestGenerator(
              step,
              this.ovmExecutionManager,
              this.ovmCallHelper,
          )
      })
  }

  getFunctionParams(): any[] {
    return [
      this.step.functionParams[0],
      this.step.functionParams[1],
      this.ovmCallHelper.interface.encodeFunctionData(
        'runSteps',
        [
          {
            callsToEM: this.getCalleeGenerators().map((calleeGenerator) => {
                return calleeGenerator.getCalldata()
            }),
            shouldRevert: !this.step.returnStatus
          }
        ]
      )
    ]
  }

  getReturnValues(): any[] {
    return [
      this.step.returnStatus,
      this.ovmCallHelper.interface.encodeFunctionResult(
      'runSteps',
        [
          this.getCalleeGenerators().map((calleeGenerator) => {
            return {
              success: true,
              data: calleeGenerator.getReturnData()
            }
          })
        ]
      )
    ]
  }
}

export const getTestGenerator = (
  step: TestStep,
  ovmExecutionManager: Contract,
  ovmCallHelper: Contract
): TestCallGenerator => {
  switch (step.functionName) {
    case 'ovmCALL':
      return new ovmCALLGenerator(
        ovmExecutionManager,
        ovmCallHelper,
        step
      )
    default:
      return new DefaultTestGenerator(
        ovmExecutionManager,
        ovmCallHelper,
        step
      )
    }
}
