import fs from 'fs'
import path from 'path'

type Check = {
  check: (name: string, parts: string[]) => void
  makeError: (name: string) => Error
}

/**
 * Function name checks
 */
const FunctionNameChecks: { [name: string]: Check } = {
  invalidCase: {
    check: (name: string, parts: string[]): void => {
      parts.forEach((part) => {
        if (part[0] !== part[0].toLowerCase()) {
          throw FunctionNameChecks.invalidCase.makeError(name)
        }
      })
    },
    makeError: (name: string): Error =>
      new Error(
        `Invalid test name "${name}".\n Test name parts should be in camelCase.`
      ),
  },
  invalidNumParts: {
    check: (name: string, parts: string[]): void => {
      if (parts.length < 3 || parts.length > 4) {
        throw FunctionNameChecks.invalidNumParts.makeError(name)
      }
    },
    makeError: (name: string): Error =>
      new Error(
        `Invalid test name "${name}".\n Test names should have either 3 or 4 parts, each separated by underscores.`
      ),
  },
  invalidPrefix: {
    check: (name: string, parts: string[]): void => {
      if (!['test', 'testFuzz', 'testDiff'].includes(parts[0])) {
        throw FunctionNameChecks.invalidPrefix.makeError(name)
      }
    },
    makeError: (name: string): Error =>
      new Error(
        `Invalid test name "${name}".\n Names should begin with "test", "testFuzz", or "testDiff".`
      ),
  },
  invalidTestResult: {
    check: (name: string, parts: string[]): void => {
      if (
        !['succeeds', 'reverts', 'fails', 'benchmark', 'works'].includes(
          parts[parts.length - 1]
        ) &&
        parts[parts.length - 2] !== 'benchmark'
      ) {
        throw FunctionNameChecks.invalidTestResult.makeError(name)
      }
    },
    makeError: (name: string): Error =>
      new Error(
        `Invalid test name "${name}".\n Test names should end with either "succeeds", "reverts", "fails", "works" or "benchmark[_num]".`
      ),
  },
  noFailureReason: {
    check: (name: string, parts: string[]): void => {
      if (
        ['reverts', 'fails'].includes(parts[parts.length - 1]) &&
        parts.length < 4
      ) {
        throw FunctionNameChecks.noFailureReason.makeError(name)
      }
    },
    makeError: (name: string): Error =>
      new Error(
        `Invalid test name "${name}".\n Failure tests should have 4 parts. The third part should indicate the reason for failure.`
      ),
  },
}

// Given a test function name, ensures it matches the expected format
const handleFunctionName = (name: string) => {
  if (!name.startsWith('test')) {
    return
  }
  const parts = name.split('_')
  Object.values(FunctionNameChecks).forEach(({ check }) => check(name, parts))
}

const main = async () => {
  const artifactsPath = './forge-artifacts'

  // Get a list of all solidity files with the extension t.sol
  const solTestFiles = fs
    .readdirSync(artifactsPath)
    .filter((solFile) => solFile.includes('.t.sol'))

  // Build a list of artifacts for contracts which include the string Test in them
  let testArtifacts: string[] = []
  for (const file of solTestFiles) {
    testArtifacts = testArtifacts.concat(
      fs
        .readdirSync(path.join(artifactsPath, file))
        .filter((x) => x.includes('Test'))
        .map((x) => path.join(artifactsPath, stf, x))
    )
  }

  for (const artifact of testArtifacts) {
    JSON.parse(fs.readFileSync(artifact, 'utf8'))
      .abi.filter((el) => el.type === 'function')
      .forEach((el) => {
        try {
          handleFunctionName(el.name)
        } catch (error) {
          throw new Error(`In ${path.parse(artifact).name}: ${error.message}`)
        }
      })
  }
}

main()
