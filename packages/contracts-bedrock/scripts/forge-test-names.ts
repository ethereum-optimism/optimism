import fs from 'fs'
import path from 'path'

type Check = {
  check: (name: string, parts: string[]) => void,
  makeError: (name: string) => Error,
}

/**
  * Function name checks
  */
const FunctionNameChecks: { [name: string]: Check } = {
  invalidCase: {
    check: (name: string, parts: string[]): void => {
      parts.forEach((part) => {
        if (part[0] !== part[0].toLowerCase())
          throw FunctionNameChecks.invalidCase.makeError(name)
      })
    },
    makeError: (name: string): Error => new Error(
      `Invalid test name: ${name}.\n Test name parts should be in camelCase.`
    )
  },
  invalidNumParts: {
    check: (name: string, parts: string[]): void => {
      if (parts.length < 3 || parts.length > 4)
        throw FunctionNameChecks.invalidNumParts.makeError(name)
    },
    makeError: (name: string): Error => new Error(
      `Invalid test name: ${name}.\n Test names should have either 3 or 4 parts, each separated by underscores.`
    )
  },
  invalidPrefix: {
    check: (name: string, parts: string[]): void => {
      if (!['test', 'testFuzz', 'testDiff'].includes(parts[0]))
        throw FunctionNameChecks.invalidPrefix.makeError(name)
    },
    makeError: (name: string): Error => new Error(
      `Invalid test name: ${name}.\n Names should begin with "test", "testFuzz", or "testDiff".`
    )
  },
  invalidTestResult: {
    check: (name: string, parts: string[]): void => {
      if (
        !['succeeds', 'reverts', 'fails', 'benchmark', 'works'].includes(
          parts[parts.length - 1]
        ) &&
        parts[parts.length - 2] !== 'benchmark'
      )
        throw FunctionNameChecks.invalidTestResult.makeError(name)
    },
    makeError: (name: string): Error => new Error(
      `Invalid test name: ${name}.\n Test names should end with either "succeeds", "reverts", "fails", "works" or "benchmark[_num]".`
    )
  },
  noFailureReason: {
    check: (name: string, parts: string[]): void => {
      if (
        ['reverts', 'fails'].includes(parts[parts.length - 1]) &&
        parts.length < 4
      )
        throw FunctionNameChecks.noFailureReason.makeError(name)
    },
    makeError: (name: string): Error => new Error(
      `Invalid test name: ${name}.\n Failure tests should have 4 parts. The third part should indicate the reason for failure.`
    )
  }
}

// Given a test function name, ensures it matches the expected format
const handleFunctionName = (name: string) => {
  if (!name.startsWith('test'))
    return
  const parts = name.split('_')
  Object.values(FunctionNameChecks).forEach(({ check }) => check(name, parts))
}

// Todo: define this function for validating contract names
// Given a test contract name, ensures it matches the expected format
const handleContractName = (name: string) => {
  name
}

const main = async () => {
  const testPath = './contracts/test'
  const testFiles = fs.readdirSync(testPath)

  for (const testFile of testFiles) {
    const filePath = path.join(testPath, testFile)
    const lines = fs
      .readFileSync(filePath, 'utf-8')
      .split('\n')
      .map((l) => l.trim())
    let currentContract: string
    for (const line of lines) {
      if (line.startsWith('contract')) {
        currentContract = line.split(' ')[1]
        handleContractName(line)
        continue
      } else if (line.startsWith('function')) {
        const funcName = line.split(' ')[1].split('(')[0]
        try {
          handleFunctionName(funcName)
        } catch (error) {
          throw new Error(
            `In ${filePath}::${currentContract}:\n ${error.message}`
          )
        }
        continue
      }
    }
  }
}

main()
