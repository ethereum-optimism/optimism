import fs from 'fs'
import path from 'path'

const testPath = './contracts/test'
const testFiles = fs.readdirSync(testPath)

const handleFunctionName = (name: string) => {
  if (!name.startsWith('test')) {
    return
  }
  const parts = name.split('_')
  parts.forEach((part) => {
    if (part[0] !== part[0].toLowerCase()) {
      throw new Error(
        `Invalid test name: ${name}. Test name parts should be in camelCase`
      )
    }
  })
  if (parts.length < 3 || parts.length > 4) {
    throw new Error(
      `Invalid test name: ${name}. Test names should have either 3 or 4 parts, each separated by underscores`
    )
  }
  if (!['test', 'testFuzz'].includes(parts[0])) {
    throw new Error(
      `Invalid test name: ${name}. Names should begin with either "test" or "testFuzz"`
    )
  }
  if (
    !['succeeds', 'reverts', 'fails', 'differential', 'benchmark'].includes(
      parts[parts.length - 1]
    ) &&
    parts[parts.length - 2] !== 'benchmark'
  ) {
    throw new Error(
      `Invalid test name: ${name}. Test names should end with either "succeeds", "reverts", "fails", "differential" or "benchmark[_num]"`
    )
  }
  if (
    ['reverts', 'fails'].includes(parts[parts.length - 1]) &&
    parts.length < 4
  ) {
    throw new Error(
      `Invalid test name: ${name}. Failure tests should have 4 parts. The third part should indicate the reason for failure.`
    )
  }
}

// Todo
const handleContractName = (name: string) => {
  name
}

for (const testFile of testFiles) {
  const lines = fs
    .readFileSync(path.join(testPath, testFile), 'utf-8')
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
          `In ${testFile}::${currentContract}:\n ${error.message}`
        )
      }
      continue
    }
  }
}
