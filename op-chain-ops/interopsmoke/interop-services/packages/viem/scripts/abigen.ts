import { execSync } from 'child_process'
import { Eta } from 'eta'
import fs from 'fs'
import path from 'path'

// Hardcoded. Can take a more elaborate approach if needed.
const OPTIMISM_PATH = path.join('..', '..', 'lib', 'optimism')
const CONTRACTS = [
  'CrossDomainMessenger',
  'CrossL2Inbox',
  'L2ToL2CrossDomainMessenger',
  'OptimismMintableERC20',
  'OptimismMintableERC20Factory',
  'StandardBridge',
  'SuperchainERC20',
  'SuperchainETHBridge',
  'SuperchainTokenBridge',
]
const OPTIMISM_CONTRACT_VERSIONS = {
  SuperchainERC20: '0.8.28',
}
const INTEROP_LIB_PATH = path.join('..', '..', 'lib', 'interop-lib')
const EXPERIMENTAL_INTEROP_LIB_CONTRACTS = ['GasTank']

/** Utility Functions */

function camelCase(str: string): string {
  const [start, ...rest] = str
  return start.toLowerCase() + rest.join('')
}

/** Abi Generation */

async function main() {
  // eslint-disable-next-line no-console
  console.log('Generating forge artifacts...')
  const contractsBedrockPath = path.join(
    OPTIMISM_PATH,
    'packages',
    'contracts-bedrock',
  )
  try {
    execSync('forge build', { cwd: contractsBedrockPath, stdio: 'inherit' })
  } catch (error) {
    throw new Error(
      `Failed to generate contracts-bedrock forge artifacts: ${error}`,
    )
  }

  const interopLibPath = path.join(INTEROP_LIB_PATH)
  try {
    execSync('forge build', { cwd: interopLibPath, stdio: 'inherit' })
  } catch (error) {
    throw new Error(`Failed to generate interop-lib forge artifacts: ${error}`)
  }

  // eslint-disable-next-line no-console
  console.log('Extracting abi generation...')
  const eta = new Eta({
    views: './scripts/templates',
    debug: true,
    autoTrim: [false, false],
  })

  const contracts = CONTRACTS.map((contract) => {
    // eslint-disable-next-line no-console
    console.log(`Generating Abi for ${contract}`)
    const abiPath = path.join(
      contractsBedrockPath,
      'forge-artifacts',
      `${contract}.sol`,
      `${contract}${
        OPTIMISM_CONTRACT_VERSIONS[contract]
          ? `.${OPTIMISM_CONTRACT_VERSIONS[contract]}`
          : ''
      }.json`,
    )
    const abi = JSON.parse(fs.readFileSync(abiPath, 'utf8')).abi
    return { name: contract, exportName: camelCase(contract), abi }
  })

  const experimentalInteropLibContracts =
    EXPERIMENTAL_INTEROP_LIB_CONTRACTS.map((contract) => {
      // eslint-disable-next-line no-console
      console.log(`Generating Abi for ${contract}`)
      const abiPath = path.join(
        interopLibPath,
        'out',
        `${contract}.sol`,
        `${contract}.json`,
      )
      const abi = JSON.parse(fs.readFileSync(abiPath, 'utf8')).abi
      return { name: contract, exportName: camelCase(contract), abi }
    })

  const fileContents = eta.render('abis', {
    contracts,
    prettyPrintJSON: (obj: any) => JSON.stringify(obj, null, 2),
  })

  fs.writeFileSync(`src/abis/index.ts`, fileContents)

  const experimentalInteropLibFileContents = eta.render('abis', {
    contracts: experimentalInteropLibContracts,
    prettyPrintJSON: (obj: any) => JSON.stringify(obj, null, 2),
  })

  fs.writeFileSync(
    `src/abis/experimental/index.ts`,
    experimentalInteropLibFileContents,
  )
}

;(async () => {
  try {
    await main()
    process.exit(0)
  } catch (e) {
    console.error(e)
    process.exit(1)
  }
})()
