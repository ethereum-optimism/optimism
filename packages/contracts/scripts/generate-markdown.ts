import fs from 'fs'
import path from 'path'

import dirtree from 'directory-tree'

import { predeploys } from '../src'

interface DeploymentInfo {
  folder: string
  name: string
  chainid: number
  rpc?: string
  l1Explorer?: string
  l2Explorer?: string
  notice?: string
}

const PUBLIC_DEPLOYMENTS: DeploymentInfo[] = [
  {
    folder: 'mainnet',
    name: 'Optimism (mainnet)',
    chainid: 10,
    rpc: 'https://mainnet.optimism.io',
    l1Explorer: 'https://etherscan.io',
    l2Explorer: 'https://optimistic.etherscan.io',
  },
  {
    folder: 'goerli',
    name: 'Optimism Goerli (public testnet)',
    chainid: 420,
    rpc: 'https://goerli.optimism.io',
    l1Explorer: 'https://goerli.etherscan.io',
    l2Explorer: 'https://goerli-optimism.etherscan.io/',
  },
]

// List of contracts that are part of a deployment but aren't meant to be used by the general
// public. E.g., implementation addresses for proxy contracts or helpers used during the
// deployment process. Although these addresses are public and users can technically try to use
// them, there's no point in doing so. As a result, we hide these addresses to avoid confusion.
const HIDDEN_CONTRACTS = [
  // Used for being able to verify the ChugSplashProxy contract.
  'L1StandardBridge_for_verification_only',
  // Implementation address for the Proxy__OVM_L1CrossDomainMessenger.
  'OVM_L1CrossDomainMessenger',
  // Utility for modifying many records in the AddressManager at the same time.
  'AddressDictator',
  // Utility for modifying a ChugSplashProxy during an upgrade.
  'ChugSplashDictator',
]

interface ContractInfo {
  name: string
  address: string
}

/**
 * Gets the full deployment folder path for a given deployment.
 *
 * @param name Deployment folder name.
 * @returns Full path to the deployment folder.
 */
const getDeploymentFolderPath = (name: string): string => {
  return path.resolve(__dirname, `../deployments/${name}`)
}

/**
 * Helper function for adding a line to a string. Avoids having to add the ugly \n to each new line
 * that you want to add a string.
 *
 * @param str String to add a line to.
 * @param line Line to add to the string.
 * @returns String with the added line and a newline at the end.
 */
const addline = (str: string, line: string): string => {
  return str + line + '\n'
}

/**
 * Generates a nicely formatted table presenting a list of contracts.
 *
 * @param contracts List of contracts to display.
 * @param explorer URL for etherscan for the network that the contracts are deployed to.
 * @returns Nicely formatted markdown-compatible table as a string.
 */
const buildContractsTable = (
  contracts: ContractInfo[],
  explorer?: string
): string => {
  // Being very verbose within this function to make it clear what's going on.
  // We use HTML instead of markdown so we can get a table that displays well on GitHub.
  // GitHub READMEs are 1012px wide. Adding a 506px image to each table header is a hack that
  // allows us to create a table where each column is 1/2 the full README width.
  let table = ``
  table = addline(table, '<table>')
  table = addline(table, '<tr>')
  table = addline(table, '<th>')
  table = addline(table, '<img width="506px" height="0px" />')
  table = addline(table, '<p><small>Contract</small></p>')
  table = addline(table, '</th>')
  table = addline(table, '<th>')
  table = addline(table, '<img width="506px" height="0px" />')
  table = addline(table, '<p><small>Address</small></p>')
  table = addline(table, '</th>')
  table = addline(table, '</tr>')

  for (const contract of contracts) {
    // Don't add records for contract addresses that aren't meant to be public-facing.
    if (HIDDEN_CONTRACTS.includes(contract.name)) {
      continue
    }

    table = addline(table, '<tr>')
    table = addline(table, '<td>')
    table = addline(table, contract.name)
    table = addline(table, '</td>')
    table = addline(table, '<td align="center">')
    if (explorer) {
      table = addline(
        table,
        `<a href="${explorer}/address/${contract.address}">`
      )
      table = addline(table, `<code>${contract.address}</code>`)
      table = addline(table, '</a>')
    } else {
      table = addline(table, `<code>${contract.address}</code>`)
    }
    table = addline(table, '</td>')
    table = addline(table, '</tr>')
  }

  table = addline(table, '</table>')
  return table
}

/**
 * Gets the list of L1 contracts for a given deployment.
 *
 * @param deployment Folder where the deployment is located.
 * @returns List of L1 contracts for thegiven deployment.
 */
const getL1Contracts = (deployment: string): ContractInfo[] => {
  const l1ContractsFolder = getDeploymentFolderPath(deployment)
  return dirtree(l1ContractsFolder)
    .children.filter((child) => {
      return child.extension === '.json'
    })
    .map((child) => {
      return {
        name: child.name.replace('.json', ''),
        // eslint-disable-next-line @typescript-eslint/no-var-requires
        address: require(path.join(l1ContractsFolder, child.name)).address,
      }
    })
}

/* eslint-disable @typescript-eslint/no-unused-vars */
/**
 * Gets the list of L2 contracts for a given deployment.
 *
 * @param deployment Folder where the deployment is located.
 * @returns List of L2 system contracts for the given deployment.
 */
const getL2Contracts = (deployment: string): ContractInfo[] => {
  // Deployment parameter is currently unused because all networks have the same predeploy
  // addresses. However, we've had situations in the past where we've had to deploy one-off
  // system contracts to a network. If we want to do that again in the future then we'll want some
  // kind of custom logic based on the network in question. Hence, the deployment parameter.
  return Object.entries(predeploys).map(([name, address]) => {
    return {
      name,
      address,
    }
  })
}
/* eslint-enable @typescript-eslint/no-unused-vars */

const main = async () => {
  for (const deployment of PUBLIC_DEPLOYMENTS) {
    let md = ``
    md = addline(md, `# ${deployment.name}`)
    if (deployment.notice) {
      md = addline(md, `## Notice`)
      md = addline(md, deployment.notice)
    }
    md = addline(md, `## Network Info`)
    md = addline(md, `- **Chain ID**: ${deployment.chainid}`)
    if (deployment.rpc) {
      md = addline(md, `- **Public RPC**: ${deployment.rpc}`)
    }
    if (deployment.l2Explorer) {
      md = addline(md, `- **Block Explorer**: ${deployment.l2Explorer}`)
    }
    md = addline(md, `## Layer 1 Contracts`)
    md = addline(
      md,
      buildContractsTable(
        getL1Contracts(deployment.folder),
        deployment.l1Explorer
      )
    )
    md = addline(md, `## Layer 2 Contracts`)
    md = addline(
      md,
      buildContractsTable(
        getL2Contracts(deployment.folder),
        deployment.l2Explorer
      )
    )

    // Write the README file for the deployment
    fs.writeFileSync(
      path.join(getDeploymentFolderPath(deployment.folder), 'README.md'),
      md
    )
  }

  let primary = ``
  primary = addline(primary, `# Optimism Deployments`)
  for (const deployment of PUBLIC_DEPLOYMENTS) {
    primary = addline(
      primary,
      `- [${deployment.name}](./${deployment.folder}#readme)`
    )
  }

  // Write the primary README file
  fs.writeFileSync(path.resolve(__dirname, '../deployments/README.md'), primary)
}

main()
