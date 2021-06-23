#!/usr/bin/env node
const dirtree = require('directory-tree')
const fs = require('fs')

/**
 *
 * takes a directory of hardhat artifacts and builds a markdown table that shows the name of the contract in one column and its address in another column with a hyperlink to etherscan
 *
 */

const networks = {
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

  let md = `# Optimism Regenesis Deployments
## LAYER 2

### Chain IDs:
- Mainnet: 10
- Kovan: 69
- Goerli: 420
*The contracts relevant for the majority of developers are \`OVM_ETH\` and the cross-domain messengers. The L2 addresses don't change.*

### Predeploy contracts:
|Contract|Address|
|--|--|
|OVM_ETH: | \`0x4200000000000000000000000000000000000006\`
|OVM_L2StandardBridge: | \`0x4200000000000000000000000000000000000010\`
|OVM_L2CrossDomainMessenger: | \`0x4200000000000000000000000000000000000007\`
|OVM_L2ToL1MessagePasser: | \`0x4200000000000000000000000000000000000000\`
|OVM_L1MessageSender: | \`0x4200000000000000000000000000000000000001\`
|OVM_DeployerWhitelist: | \`0x4200000000000000000000000000000000000002\`
|OVM_ECDSAContractAccount: | \`0x4200000000000000000000000000000000000003\`
|OVM_SequencerEntrypoint: | \`0x4200000000000000000000000000000000000005\`
|Lib_AddressManager: | \`0x4200000000000000000000000000000000000008\`
|ERC1820Registry: | \`0x1820a4B7618BdE71Dce8cdc73aAB6C95905faD24\`

---
---

## LAYER 1\n\n`

  for (const deployment of deployments) {
    md += `## ${deployment.toUpperCase()}\n\n`

    const chainId = Number(
      fs.readFileSync(`./deployments/${deployment}/.chainId`)
    )
    const network = networks[chainId]

    md += `Network : __${network} (chain id: ${chainId})__\n\n`

    md += `|Contract|Address|\n`
    md += `|--|--|\n`

    const contracts = dirtree(`./deployments/${deployment}`)
      .children.filter((child) => {
        return child.extension === '.json'
      })
      .map((child) => {
        return child.name.replace('.json', '')
      })

    proxiedContracts = []
    for (let i = 0; i < contracts.length; i++) {
      if (contracts[i] == 'OVM_L1CrossDomainMessenger') {
        proxiedContracts.push(contracts.splice(i, 1)[0])
      }
      if (contracts[i] == 'OVM_L1ETHGateway') {
        proxiedContracts.push(contracts.splice(i, 1)[0])
      }
    }

    for (const contract of contracts) {
      const colonizedName = contract.split(':').join('-')

      const deploymentInfo = require(`../deployments/${deployment}/${contract}.json`)

      const escPrefix = chainId !== 1 ? `${network}.` : ''
      const etherscanUrl = `https://${escPrefix}etherscan.io/address/${deploymentInfo.address}`
      md += `|${colonizedName}|[${deploymentInfo.address}](${etherscanUrl})|\n`
    }

    md += `<!--\nImplementation addresses. DO NOT use these addresses directly.\nUse their proxied counterparts seen above.\n\n`

    for (const proxy of proxiedContracts) {
      const colonizedName = proxy.split(':').join('-')

      const deploymentInfo = require(`../deployments/${deployment}/${proxy}.json`)

      const escPrefix = chainId !== 1 ? `${network}.` : ''
      const etherscanUrl = `https://${escPrefix}etherscan.io/address/${deploymentInfo.address}`
      md += `${colonizedName}: \n - ${deploymentInfo.address}\n - ${etherscanUrl})\n`
    }

    md += `-->\n`
    md += `---\n`
  }

  fs.writeFileSync(`./deployments/README.md`, md)
})().catch(console.error)
