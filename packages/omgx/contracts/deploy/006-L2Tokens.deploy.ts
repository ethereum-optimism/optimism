/* Imports: External */
 import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
 import { Contract, ContractFactory} from 'ethers'
 import { getContractFactory } from '@eth-optimism/contracts'
 import chalk from 'chalk'
 import * as fs from 'fs'
 import path from 'path'

 let Factory__L2ERC20: ContractFactory

 let L2ERC20: Contract

 const deployFn: DeployFunction = async (hre) => {

     const supportList = path.resolve(__dirname, "../preSupportedTokens.json")
     
     if (fs.existsSync(supportList)) {
         const supportedTokenData = fs.readFileSync(supportList, 'utf8')
         const supportedTokensArray = JSON.parse(supportedTokenData).supportedTokens


         Factory__L2ERC20 = getContractFactory(
             "L2StandardERC20",
             (hre as any).deployConfig.deployer_l2,
             true,
         )

         console.log('Deploying L2 ERC20s..')
         let deployments = {}
         for (const token of supportedTokensArray) {
             L2ERC20 = await Factory__L2ERC20.deploy(
                 (hre as any).deployConfig.L2StandardBridgeAddress,
                 token.address,
                 token.name,
                 token.symbol,
                 {gasLimit: 800000, gasPrice: 0}
             )
             await L2ERC20.deployTransaction.wait()
             console.log(`🌕 ${chalk.red(`L2 ${token.symbol} deployed to:`)} ${chalk.green(L2ERC20.address)}`)
             deployments[token.symbol] = L2ERC20.address
         }

         const dumpsPath = path.resolve(__dirname, "../dist/dumps")
         if (!fs.existsSync(dumpsPath)) {
           fs.mkdirSync(dumpsPath, { recursive: true })
         }
         const addrsPath = path.resolve(dumpsPath, 'l2TokenAddresses.json')
         fs.writeFileSync(addrsPath, JSON.stringify(deployments, null, 2))
     }
 }

 deployFn.tags = ['ERC20Tokens', 'optional']

 export default deployFn 