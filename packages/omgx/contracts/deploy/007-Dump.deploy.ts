/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import path from 'path'
import dirtree from 'directory-tree'
import fs from 'fs'

const deployFn: DeployFunction = async (hre) => {

    let contracts = {};
    
    const deployments = await hre.deployments.all()
        
    for (let key in await hre.deployments.all()) {
      contracts[key] = deployments[key].address
    }
    
    const addresses = JSON.stringify(contracts, null, 2)
    
    const dumpsPath = path.resolve(__dirname, "../dist/dumps")
    
    if (!fs.existsSync(dumpsPath)) {
      fs.mkdirSync(dumpsPath, { recursive: true })
    }
    const addrsPath = path.resolve(dumpsPath, 'addresses.json')
    fs.writeFileSync(addrsPath, addresses)

}

deployFn.tags = ['Log', 'required']

export default deployFn
