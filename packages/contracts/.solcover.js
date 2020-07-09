/**
 * SolCover configuration management:
 * 
 * SolCover unfortunately doesn't provide us with the ability to exclusively
 * generate coverage reports for a given file. Although we can use the
 * `--testfiles`` parameter to limit tests to a particular glob, SolCover will
 * still try to generate coverage reports for anything not covered within the
 * `skipFiles` option exported below. `skipFiles` additionally does not parse
 * globs, creating a mismatch between it and the `--testfiles` option.
 * 
 * To address the above issues, we take the following steps:
 * 1. Parse the `--testfiles` option from our command-line arguments.
 * 2. Use the `--testfiles` option to find the list of contracts to be tested.
 * 3. Find *all* contracts and exclude the results of (2).
 * 4. Add the result of (3) to `skipFiles`.
 * 
 * NOTE: The above will *only* work if contract test files follow the
 * `<ContractName>.spec.ts` convention. Our function will fail to find the
 * correct contracts otherwise.
 */ 

const path = require('path')
const glob = require('glob')
const ArgumentParser = require('argparse').ArgumentParser

const parser = new ArgumentParser()
parser.addArgument(
  ['--testfiles'],
)

/**
 * Given a glob, finds all files to skip.
 * @param skipFiles Path or glob of files to skip.
 * @returns Paths to files to skip.
 */
const parseSkipFiles = (skipFiles) => {
  return skipFiles.reduce((files, contract) => {
    return files.concat(glob.sync(contract))
  }, []).map((contract) => {
    return contract.replace('contracts/', '')
  })
}

/**
 * Given a glob, finds all contracts to skip that are *not* the given files.
 * @param includeFiles Path or glob of files not to skip.
 * @returns Paths to files to skip.
 */
const parseIncludeFiles = (includeFiles) => {
  const allFiles = parseSkipFiles(['contracts/**/*.sol'])
  const includedFiles = parseSkipFiles(includeFiles)
  return allFiles.filter((file) => {
    return !includedFiles.includes(file)
  })
}

/**
 * Parses command-line arguments and picks out the correct contracts to skip.
 */
const parseTestFiles = () => {
  const args = parser.parseKnownArgs()[0]
  const testFileName = path.basename(args.testfiles)
  const contractFileName = testFileName.split('.')[0] + '.sol'
  return parseIncludeFiles(['contracts/**/' + contractFileName])
}

module.exports = {
  skipFiles: parseTestFiles()
}