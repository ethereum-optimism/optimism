import fs from 'fs'

import { glob } from 'glob'

/**
 * Series of function name checks.
 */
const checks: Array<{
  check: (parts: string[]) => boolean
  error: string
}> = [
  {
    error: 'test name parts should be in camelCase',
    check: (parts: string[]): boolean => {
      return parts.every((part) => {
        return part[0] === part[0].toLowerCase()
      })
    },
  },
  {
    error:
      'test names should have either 3 or 4 parts, each separated by underscores',
    check: (parts: string[]): boolean => {
      return parts.length === 3 || parts.length === 4
    },
  },
  {
    error: 'test names should begin with "test", "testFuzz", or "testDiff"',
    check: (parts: string[]): boolean => {
      return ['test', 'testFuzz', 'testDiff'].includes(parts[0])
    },
  },
  {
    error:
      'test names should end with either "succeeds", "reverts", "fails", "works" or "benchmark[_num]"',
    check: (parts: string[]): boolean => {
      return (
        ['succeeds', 'reverts', 'fails', 'benchmark', 'works'].includes(
          parts[parts.length - 1]
        ) ||
        (parts[parts.length - 2] === 'benchmark' &&
          !isNaN(parseInt(parts[parts.length - 1], 10)))
      )
    },
  },
  {
    error:
      'failure tests should have 4 parts, third part should indicate the reason for failure',
    check: (parts: string[]): boolean => {
      return (
        parts.length === 4 ||
        !['reverts', 'fails'].includes(parts[parts.length - 1])
      )
    },
  },
]

/**
 * Script for checking that all test functions are named correctly.
 */
const main = async () => {
  const errors: string[] = []
  const files = glob.sync('./forge-artifacts/**/*.t.sol/*Test*.json')
  for (const file of files) {
    const artifact = JSON.parse(fs.readFileSync(file, 'utf8'))
    for (const element of artifact.abi) {
      // Skip non-functions and functions that don't start with "test".
      if (element.type !== 'function' || !element.name.startsWith('test')) {
        continue
      }

      // Check the rest.
      for (const { check, error } of checks) {
        if (!check(element.name.split('_'))) {
          errors.push(`in ${file} function ${element.name}: ${error}`)
        }
      }
    }
  }

  if (errors.length > 0) {
    console.error(...errors)
    process.exit(1)
  }
}

main()
