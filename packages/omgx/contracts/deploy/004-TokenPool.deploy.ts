/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory, utils} from 'ethers'
import chalk from 'chalk';

import L2TokenPoolJson from '../artifacts-ovm/contracts/TokenPool.sol/TokenPool.json'

let Factory__L2TokenPool: ContractFactory

let L2TokenPool: Contract


const deployFn: DeployFunction = async (hre) => {

    Factory__L2TokenPool = new ContractFactory(
      L2TokenPoolJson.abi,
      L2TokenPoolJson.bytecode,
      (hre as any).deployConfig.deployer_l2
    )

    const L2JLKN = await hre.deployments.getOrNull('L2JLKN');

    //Deploy L2 token pool for the new token
    L2TokenPool = await Factory__L2TokenPool.deploy({gasLimit: 1000000, gasPrice: 0})
    await L2TokenPool.deployTransaction.wait()
    const L2TokenPoolDeploymentSubmission: DeploymentSubmission = {
      ...L2TokenPool,
      receipt: L2TokenPool.receipt,
      address: L2TokenPool.address,
      abi: L2TokenPoolJson.abi,
    };
    await hre.deployments.save('L2TokenPool', L2TokenPoolDeploymentSubmission)
    console.log(`🌕 ${chalk.red('L2TokenPool deployed to:')} ${chalk.green(L2TokenPool.address)}`)

    if(L2JLKN === undefined){
      console.log(`!!! ${chalk.red('L2TokenPool was not registered because L2JLKN was not deployed')}`)
    }
    else{
      //Register ERC20 token address in L2 token pool
      const registerL2TokenPoolTX = await L2TokenPool.registerTokenAddress(
        L2JLKN.address,
        {gasLimit: 800000, gasPrice: 0}
      );
      await registerL2TokenPoolTX.wait()
      console.log(`⭐️ ${chalk.blue('L2TokenPool registered:')} ${chalk.green(registerL2TokenPoolTX.hash)}`)
    }


}

deployFn.tags = ['TokenPool', 'required']

export default deployFn
