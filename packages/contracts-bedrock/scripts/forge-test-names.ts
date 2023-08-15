import fs from 'fs'
import path from 'path'
import { execSync } from 'child_process'

type Check = (parts: string[]) => boolean
type Checks = Array<{
  check: Check
  error: string
}>

/**
 * Series of function name checks.
 */
const checks: Checks = [
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
  const result = execSync('forge config --json')
  const config = JSON.parse(result.toString())
  const out = config.out || 'out'

  const paths = []

  const readFilesRecursively = (dir: string) => {
    const files = fs.readdirSync(dir)

    for (const file of files) {
      const filePath = path.join(dir, file)
      const fileStat = fs.statSync(filePath)

      if (fileStat.isDirectory()) {
        readFilesRecursively(filePath)
      } else {
        paths.push(filePath)
      }
    }
  }

  readFilesRecursively(out)

  console.log('Success:')
  const errors: string[] = []

  for (const filepath of paths) {
    const artifact = JSON.parse(fs.readFileSync(filepath, 'utf8'))

    let isTest = false
    for (const element of artifact.abi) {
      if (element.name === 'IS_TEST') {
        isTest = true
        break
      }
    }

    if (isTest) {
      let success = true
      for (const element of artifact.abi) {
        // Skip non-functions and functions that don't start with "test".
        if (element.type !== 'function' || !element.name.startsWith('test')) {
          continue
        }

        // Check the rest.
        for (const { check, error } of checks) {
          if (!check(element.name.split('_'))) {
            errors.push(`${filepath}#${element.name}: ${error}`)
            success = false
          }
        }
      }
      if (success) {
        console.log(` - ${path.parse(filepath).name}`)
      }
    }
  }

  if (errors.length > 0) {
    console.error(errors.join('\n'))
    process.exit(1)
  }
}

main()
