/* External Imports */
import { BigNumber } from 'ethers'

export type ContextOpcode =
  | 'ovmCALLER'
  | 'ovmNUMBER'
  | 'ovmADDRESS'
  | 'ovmL1TXORIGIN'
  | 'ovmL1QUEUEORIGIN'
  | 'ovmTIMESTAMP'
  | 'ovmGASLIMIT'
  | 'ovmCHAINID'
  | 'ovmGETNONCE'

type CallOpcode = 'ovmCALL' | 'ovmSTATICCALL' | 'ovmDELEGATECALL'

type RevertFlagError = {
  flag: number
  nuisanceGasLeft?: number
  ovmGasRefund?: number
  data?: string
  onlyValidateFlag?: boolean
}

interface TestStep_evm {
  functionName: 'evmRETURN' | 'evmREVERT' | 'evmINVALID'
  returnData?: string | RevertFlagError
}

interface TestStep_Context {
  functionName: ContextOpcode
  expectedReturnValue: string | number | BigNumber
}

interface TestStep_REVERT {
  functionName: 'ovmREVERT'
  revertData?: string
  expectedReturnStatus: boolean
  expectedReturnValue?: string | RevertFlagError
}

interface TestStep_EXTCODESIZE {
  functionName: 'ovmEXTCODESIZE'
  functionParams: {
    address: string
  }
  expectedReturnStatus: boolean
  expectedReturnValue: number | RevertFlagError
}

interface TestStep_EXTCODEHASH {
  functionName: 'ovmEXTCODEHASH'
  functionParams: {
    address: string
  }
  expectedReturnStatus: boolean
  expectedReturnValue: string | RevertFlagError
}

interface TestStep_EXTCODECOPY {
  functionName: 'ovmEXTCODECOPY'
  functionParams: {
    address: string
    offset: number
    length: number
  }
  expectedReturnStatus: boolean
  expectedReturnValue: string | RevertFlagError
}

interface TestStep_SSTORE {
  functionName: 'ovmSSTORE'
  functionParams: {
    key: string
    value: string
  }
  expectedReturnStatus: boolean
  expectedReturnValue?: RevertFlagError
}

interface TestStep_SLOAD {
  functionName: 'ovmSLOAD'
  functionParams: {
    key: string
  }
  expectedReturnStatus: boolean
  expectedReturnValue: string | RevertFlagError
}

interface TestStep_INCREMENTNONCE {
  functionName: 'ovmINCREMENTNONCE'
  expectedReturnStatus: boolean
  expectedReturnValue?: RevertFlagError
}

export interface TestStep_CALL {
  functionName: CallOpcode
  functionParams: {
    gasLimit: number | BigNumber
    target: string
    calldata?: string
    subSteps?: TestStep[]
  }
  expectedReturnStatus: boolean
  expectedReturnValue?:
    | string
    | RevertFlagError
    | { ovmSuccess: boolean; returnData: string }
}

interface TestStep_CREATE {
  functionName: 'ovmCREATE'
  functionParams: {
    bytecode?: string
    subSteps?: TestStep[]
  }
  expectedReturnStatus: boolean
  expectedReturnValue:
    | string
    | {
        address: string
        revertData: string
      }
    | RevertFlagError
}

interface TestStep_CREATE2 {
  functionName: 'ovmCREATE2'
  functionParams: {
    salt: string
    bytecode?: string
    subSteps?: TestStep[]
  }
  expectedReturnStatus: boolean
  expectedReturnValue:
    | string
    | {
        address: string
        revertData: string
      }
    | RevertFlagError
}

interface TestStep_CREATEEOA {
  functionName: 'ovmCREATEEOA'
  functionParams: {
    _messageHash: string
    _v: number
    _r: string
    _s: string
  }
  expectedReturnStatus: boolean
  expectedReturnValue: string | RevertFlagError
}

export interface TestStep_Run {
  functionName: 'run'
  suppliedGas?: number
  functionParams: {
    timestamp: number
    queueOrigin: number
    entrypoint: string
    origin: string
    msgSender: string
    gasLimit: number
    data?: string
    subSteps?: TestStep[]
  }
  expectedRevertValue?: string
}

export interface TestStep_SETCODE {
  functionName: 'ovmSETCODE'
  functionParams: {
    address: string
    code: string
  }
  expectedReturnStatus: boolean
  expectedReturnValue?: RevertFlagError
}

export interface TestStep_SETSTORAGE {
  functionName: 'ovmSETSTORAGE'
  functionParams: {
    address: string
    key: string
    value: string
  }
  expectedReturnStatus: boolean
  expectedReturnValue?: RevertFlagError
}

export type TestStep =
  | TestStep_Context
  | TestStep_SSTORE
  | TestStep_SLOAD
  | TestStep_INCREMENTNONCE
  | TestStep_CALL
  | TestStep_CREATE
  | TestStep_CREATE2
  | TestStep_CREATEEOA
  | TestStep_EXTCODESIZE
  | TestStep_EXTCODEHASH
  | TestStep_EXTCODECOPY
  | TestStep_REVERT
  | TestStep_evm
  | TestStep_SETCODE
  | TestStep_SETSTORAGE

export interface ParsedTestStep {
  functionName: string
  functionData: string
  expectedReturnStatus: boolean
  expectedReturnData: string
  onlyValidateFlag: boolean
}

export const isRevertFlagError = (
  expectedReturnValue: any
): expectedReturnValue is RevertFlagError => {
  return (
    typeof expectedReturnValue === 'object' &&
    expectedReturnValue !== null &&
    expectedReturnValue.flag !== undefined
  )
}

export const isTestStep_evm = (step: TestStep): step is TestStep_evm => {
  return ['evmRETURN', 'evmREVERT', 'evmINVALID'].includes(step.functionName)
}

export const isTestStep_Context = (
  step: TestStep
): step is TestStep_Context => {
  return [
    'ovmCALLER',
    'ovmNUMBER',
    'ovmADDRESS',
    'ovmNUMBER',
    'ovmL1TXORIGIN',
    'ovmTIMESTAMP',
    'ovmGASLIMIT',
    'ovmCHAINID',
    'ovmL1QUEUEORIGIN',
    'ovmGETNONCE',
  ].includes(step.functionName)
}

export const isTestStep_SSTORE = (step: TestStep): step is TestStep_SSTORE => {
  return step.functionName === 'ovmSSTORE'
}

export const isTestStep_SLOAD = (step: TestStep): step is TestStep_SLOAD => {
  return step.functionName === 'ovmSLOAD'
}

export const isTestStep_INCREMENTNONCE = (
  step: TestStep
): step is TestStep_INCREMENTNONCE => {
  return step.functionName === 'ovmINCREMENTNONCE'
}

export const isTestStep_EXTCODESIZE = (
  step: TestStep
): step is TestStep_EXTCODESIZE => {
  return step.functionName === 'ovmEXTCODESIZE'
}

export const isTestStep_EXTCODEHASH = (
  step: TestStep
): step is TestStep_EXTCODEHASH => {
  return step.functionName === 'ovmEXTCODEHASH'
}

export const isTestStep_EXTCODECOPY = (
  step: TestStep
): step is TestStep_EXTCODECOPY => {
  return step.functionName === 'ovmEXTCODECOPY'
}

export const isTestStep_REVERT = (step: TestStep): step is TestStep_REVERT => {
  return step.functionName === 'ovmREVERT'
}

export const isTestStep_CALL = (step: TestStep): step is TestStep_CALL => {
  return ['ovmCALL', 'ovmSTATICCALL', 'ovmDELEGATECALL'].includes(
    step.functionName
  )
}

export const isTestStep_CREATE = (step: TestStep): step is TestStep_CREATE => {
  return step.functionName === 'ovmCREATE'
}

export const isTestStep_CREATEEOA = (
  step: TestStep
): step is TestStep_CREATEEOA => {
  return step.functionName === 'ovmCREATEEOA'
}

export const isTestStep_CREATE2 = (
  step: TestStep
): step is TestStep_CREATE2 => {
  return step.functionName === 'ovmCREATE2'
}

export const isTestStep_SETCODE = (
  step: TestStep | TestStep_Run
): step is TestStep_SETCODE => {
  return step.functionName === 'ovmSETCODE'
}

export const isTestStep_SETSTORAGE = (
  step: TestStep | TestStep_Run
): step is TestStep_SETSTORAGE => {
  return step.functionName === 'ovmSETSTORAGE'
}

export const isTestStep_Run = (
  step: TestStep | TestStep_Run
): step is TestStep_Run => {
  return step.functionName === 'run'
}

interface TestState {
  ExecutionManager: any
  StateManager: any
}

export interface TestParameter {
  name: string
  steps: Array<TestStep | TestStep_Run>
  expectInvalidStateAccess?: boolean
  focus?: boolean
  skip?: boolean
}

export interface TestDefinition {
  name: string
  focus?: boolean
  preState?: Partial<TestState>
  postState?: Partial<TestState>
  parameters?: TestParameter[]
  subTests?: TestDefinition[]
}
