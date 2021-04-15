import dirtree from 'directory-tree'
import * as fs from 'fs'

/**
 *
 * takes a directory of hardhat artifacts and builds a markdown table that shows the name of the contract in one column and its address in another column with a hyperlink to etherscan
 *
 */

const NETWORKS_NAME = {
  1: 'mainnet',
  3: 'ropsten',
  4: 'rinkeby',
  5: 'goerli',
  42: 'kovan',
}

;(async () => {
  console.log(`Writing contract addresses`)

  const deployments = dirtree(`./deployments`)
    .children.filter((child) => {
      return child.type === 'directory'
    })
    .map((d) => d.name)
    .reverse()

  let md = ''

  md += '# Optimism Regenesis\n'
  md += '## Deployments\n'

  for (const deployment of deployments) {
    md += `### ${deployment}\n`

    const chainId = Number(
      fs.readFileSync(`./deployments/${deployment}/.chainId`)
    )
    const network = NETWORKS_NAME[chainId]

    md += `Network : __${network} (${chainId})__\n`

    md += `|Contract|Address|Etherscan|\n`
    md += `|--|--|--|\n`

    const contracts = dirtree(`./deployments/${deployment}`)
      .children.filter((child) => {
        return child.extension === '.json'
      })
      .map((child) => {
        return child.name.replace('.json', '')
      })

    for (const contract of contracts) {
      const deploymentInfo = JSON.parse(
        fs
          .readFileSync(`./deployments/${deployment}/${contract}.json`)
          .toString()
      )

      const escPrefix = chainId !== 1 ? `${network}.` : ''
      const etherscanUrl = `https://${escPrefix}etherscan.io/address/${deploymentInfo.address}`
      md += `|${contract}|${deploymentInfo.address}|[Open](${etherscanUrl})|\n`
    }
  }

  fs.writeFileSync(`./addresses.md`, md)
})().catch(console.error)
