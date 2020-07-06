const path = require('path')
const glob = require('glob')
const ArgumentParser = require('argparse').ArgumentParser

const parser = new ArgumentParser()
parser.addArgument(
  ['--testfiles'],
)

const parseSkipFiles = (skipFiles) => {
  return skipFiles.reduce((files, contract) => {
    return files.concat(glob.sync(contract))
  }, []).map((contract) => {
    return contract.replace('contracts/', '')
  })
}

const parseIncludeFiles = (includeFiles) => {
  const allFiles = parseSkipFiles(['contracts/**/*.sol'])
  const includedFiles = parseSkipFiles(includeFiles)
  return allFiles.filter((file) => {
    return !includedFiles.includes(file)
  })
}

const parseTestFiles = () => {
  const args = parser.parseKnownArgs()[0]
  const testFileName = path.basename(args.testfiles)
  const contractFileName = testFileName.split('.')[0] + '.sol'
  return parseIncludeFiles(['contracts/**/' + contractFileName])
}

module.exports = {
  skipFiles: parseTestFiles()
}