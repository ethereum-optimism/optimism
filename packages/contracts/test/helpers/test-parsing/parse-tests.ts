import { expect } from '../../setup'

import { Contract, BigNumber } from "ethers"
import { TestCallGenerator, ovmADDRESSTest, ovmCALLERTest, ovmCALLTest } from "./call-generator"
import { GAS_LIMIT } from '../constants'

type SolidityFunctionParameter = string | number | BigNumber

interface TestStep {
  functionName: string
  functionParams: Array<SolidityFunctionParameter | TestStep[]>
  returnStatus: boolean
  returnValues: any[]
}

interface TestParameters {
  steps: TestStep[]
}

interface TestDefinition {
  name: string
  preState?: {
    ExecutionManager?: any,
    StateManager?: any
  },
  parameters: Array<TestParameters | TestDefinition>,
  postState?: {
    ExecutionManager?: any,
    StateManager?: any
  },
}

const isTestDefinition = (parameters: TestParameters | TestDefinition): parameters is TestDefinition => {
  return (parameters as TestDefinition).name !== undefined
}

/*
const encodeTestStep = (
  step: TestStep,
  ovmExecutionManager: Contract,
  helperCodeContract: Contract
): string => {
  const params = step.functionParams.map((functionParam) => {
    if (Array.isArray(functionParam)) {
      return 
    }
  })

  return ovmExecutionManager.interface.encodeFunctionData(
    step.functionName,
    params
  )
}
*/

const getCorrectGenerator = (step: TestStep): TestCallGenerator => {
  switch (step.functionName) {
    case 'ovmADDRESS':
      return new ovmADDRESSTest(step.returnValues[0])
    case 'ovmCALLER':
      return new ovmCALLERTest(step.returnValues[0])
    case 'ovmCALL':
      return new ovmCALLTest(
        step.functionParams[1] as string,
        (step.functionParams[2] as TestStep[]).map((param) => {
          return getCorrectGenerator(param)
        }),
        step.returnStatus,
        step.functionParams[0] as number
      )
    default:
      throw new Error('Input type not implemented.')
  }
}

const getTestGenerator = (
  step: TestStep
): TestCallGenerator => {
  return getCorrectGenerator(step)
}

export const runExecutionManagerTest = (
  test: TestDefinition,
  ovmExecutionManager: Contract,
  ovmStateManager: Contract
): void => {
  test.preState = test.preState || {}
  test.postState = test.postState || {}

  describe(`Standard test: ${test.name}`, () => {
    test.parameters.map((parameters) => {
      if (isTestDefinition(parameters)) {
        runExecutionManagerTest(
          {
            ...parameters,
            preState: {
              ...test.preState,
              ...parameters.preState
            },
            postState: {
              ...test.postState,
              ...parameters.postState
            }
          },
          ovmExecutionManager,
          ovmStateManager,
        )
      } else {
        beforeEach(async () => {
          await ovmExecutionManager.__setContractStorage(test.preState.ExecutionManager)
          await ovmStateManager.__setContractStorage(test.preState.StateManager)
        })

        afterEach(async () => {
          await ovmExecutionManager.__checkContractStorage({
            ...test.preState.ExecutionManager,
            ...test.postState.ExecutionManager
          })
          await ovmStateManager.__checkContractStorage({
            ...test.preState.StateManager,
            ...test.postState.StateManager
          })
        })

        parameters.steps.map((step, idx) => {
          it(`should run test: ${test.name} ${idx}`, async () => {
            const testGenerator = getTestGenerator(step)

            const callResult = await ovmExecutionManager.provider.call({
              to: ovmExecutionManager.address,
              data: testGenerator.generateCalldata(),
              gasLimit: GAS_LIMIT
            })
    
            await ovmExecutionManager.signer.sendTransaction({
              to: ovmExecutionManager.address,
              data: testGenerator.generateCalldata(),
              gasLimit: GAS_LIMIT
            })

            expect(callResult).to.equal(testGenerator.generateExpectedReturnData())
          })
        })
      }
    })
  })
}
