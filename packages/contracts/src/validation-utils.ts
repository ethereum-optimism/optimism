import { createInterface } from 'readline'
import { hexStringEquals } from '@eth-optimism/core-utils'

export const getInput = (query) => {
  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  })

  return new Promise((resolve) =>
    rl.question(query, (ans) => {
      rl.close()
      resolve(ans)
    })
  )
}

const codes = {
  reset: '\x1b[0m',
  red: '\x1b[0;31m',
  green: '\x1b[0;32m',
  cyan: '\x1b[0;36m',
  yellow: '\x1b[1;33m',
}

export const color = Object.fromEntries(
  Object.entries(codes).map(([k]) => [
    k,
    (msg: string) => `${codes[k]}${msg}${codes.reset}`,
  ])
)

// helper for finding the right artifact from the deployed name
const locateArtifact = (name: string) => {
  return {
    'ChainStorageContainer-CTC-batches':
      'L1/rollup/ChainStorageContainer.sol/ChainStorageContainer.json',
    'ChainStorageContainer-SCC-batches':
      'L1/rollup/ChainStorageContainer.sol/ChainStorageContainer.json',
    CanonicalTransactionChain:
      'L1/rollup/CanonicalTransactionChain.sol/CanonicalTransactionChain.json',
    StateCommitmentChain:
      'L1/rollup/StateCommitmentChain.sol/StateCommitmentChain.json',
    BondManager: 'L1/verification/BondManager.sol/BondManager.json',
    OVM_L1CrossDomainMessenger:
      'L1/messaging/L1CrossDomainMessenger.sol/L1CrossDomainMessenger.json',
    Proxy__OVM_L1CrossDomainMessenger:
      'libraries/resolver/Lib_ResolvedDelegateProxy.sol/Lib_ResolvedDelegateProxy.json',
    Proxy__OVM_L1StandardBridge:
      'chugsplash/L1ChugSplashProxy.sol/L1ChugSplashProxy.json',
  }[name]
}

export const getArtifactFromManagedName = (name: string) => {
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  return require(`../artifacts/contracts/${locateArtifact(name)}`)
}

export const getEtherscanUrl = (network, address: string) => {
  const escPrefix = network.chainId !== 1 ? `${network.name}.` : ''
  return `https://${escPrefix}etherscan.io/address/${address}`
}

// Reduces a byte string to first 32 bytes, with a '...' to indicate when it was shortened
const truncateLongString = (value: string): string => {
  return value.length > 66 ? `${value.slice(0, 66)}...` : value
}

export const printSectionHead = (msg: string) => {
  console.log(color.cyan(msg))
  console.log(
    color.cyan('='.repeat(Math.max(...msg.split('\n').map((s) => s.length))))
  )
}

export const printComparison = (
  action: string,
  description: string,
  expected: { name: string; value: any },
  deployed: { name: string; value: any }
) => {
  console.log(`\n${action}:`)
  if (hexStringEquals(expected.value, deployed.value)) {
    console.log(
      color.green(`${expected.name}: ${truncateLongString(expected.value)}`)
    )
    console.log('matches')
    console.log(
      color.green(`${deployed.name}: ${truncateLongString(deployed.value)}`)
    )
    console.log(color.green(`${description} looks good! 😎`))
  } else {
    throw new Error(
      `${description} looks wrong. ${expected.value}\ndoes not match\n${deployed.value}.`
    )
  }
}
