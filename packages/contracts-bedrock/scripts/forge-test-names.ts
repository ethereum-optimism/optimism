import fs from 'fs'
import path from 'path'

const testPath = './contracts/test'
const testFiles = fs.readdirSync(testPath)

// Given a test function name, ensures it matches the expected format
const handleFunctionName = (name: string) => {
  if (!name.startsWith('test')) {
    return
  }
  const parts = name.split('_')
  parts.forEach((part) => {
    // Good enough approximation for camelCase
    if (part[0] !== part[0].toLowerCase()) {
      throw new Error(
        `Invalid test name: ${name}.\n Test name parts should be in camelCase`
      )
    }
  })
  if (parts.length < 3 || parts.length > 4) {
    throw new Error(
      `Invalid test name: ${name}.\n Test names should have either 3 or 4 parts, each separated by underscores`
    )
  }
  if (!['test', 'testFuzz', 'testDiff'].includes(parts[0])) {
    throw new Error(
      `Invalid test name: ${name}.\n Names should begin with either "test" or "testFuzz"`
    )
  }
  if (
    !['succeeds', 'reverts', 'fails', 'benchmark', 'works'].includes(
      parts[parts.length - 1]
    ) &&
    parts[parts.length - 2] !== 'benchmark'
  ) {
    throw new Error(
      `Invalid test name: ${name}.\n Test names should end with either "succeeds", "reverts", "fails", "differential" or "benchmark[_num]"`
    )
  }
  if (
    ['reverts', 'fails'].includes(parts[parts.length - 1]) &&
    parts.length < 4
  ) {
    throw new Error(
      `Invalid test name: ${name}.\n Failure tests should have 4 parts. The third part should indicate the reason for failure.`
    )
  }
}

// Todo: define this function for validating contract names
// Given a test contract name, ensures it matches the expected format
const handleContractName = (name: string) => {
  name
}

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
