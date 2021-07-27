const fs = require("fs");
const dotenv = require('dotenv');
const bre = require("hardhat");
const { utils } = require('ethers');
dotenv.config();
const { deploy } = require('./utils');

const env = process.env;
const deployPrivateKey = env.DEPLOYER_PRIVATE_KEY;
const l2RpcUrl = env.L2_NODE_WEB3_URL;
const l2ETHAddress = "0x4200000000000000000000000000000000000006";

gasSettings = {
  gasLimit: 60000000, 
  gasPrice: 15000000
}

function writeFileSyncRecursive(filename, content, charset) {
  const folders = filename.split('/').slice(0, -1)
  if (folders.length) {
    // create folder path if it doesn't exist
    folders.reduce((last, folder) => {
      const folderPath = last ? last + '/' + folder : folder
      if (!fs.existsSync(folderPath)) {
        fs.mkdirSync(folderPath, { recursive: true })
      }
      return folderPath
    })
  }
  fs.writeFileSync('.'+filename, content, charset)
}

const main = async () => {

  console.log(` 📡 Deploying...\n`);

  const deployAddress = new ethers.Wallet(deployPrivateKey).address;

  // contracts
  const SushiToken = await deploy({
    contractName: "SushiToken",
    rpcUrl: l2RpcUrl,
    pk: deployPrivateKey,
    ovm: true,
    _args: []
  })

  console.log(await SushiToken.symbol())

  const SushiBar = await deploy({
    contractName: "SushiBar",
    rpcUrl: l2RpcUrl,
    pk: deployPrivateKey,
    ovm: true,
    _args: []
  })

  await (await SushiBar.initialize(
    SushiToken.address, 
    gasSettings
   )).wait()

  const MasterChef = await deploy({
    contractName: "MasterChef",
    rpcUrl: l2RpcUrl,
    pk: deployPrivateKey,
    ovm: true,
    _args: []
  });

  await (await MasterChef.initialize(
    SushiToken.address, 
    deployAddress, 
    utils.parseEther("100000000"), 
    "0", 
    utils.parseEther("100000000"), 
    gasSettings
  )).wait()
  console.log(SushiToken.address)
  
  if (await SushiToken.owner() !== MasterChef.address) {
    // Transfer Sushi Ownership to Chef
    console.log(" 🔑 Transfer Sushi Ownership to Chef")
    await (await SushiToken.transferOwnership(
      MasterChef.address, 
      gasSettings
    )).wait()
  }

  if (await MasterChef.owner() !== deployAddress) {
    // Transfer ownership of MasterChef to Dev
    console.log(" 🔑 Transfer ownership of MasterChef to Dev")
    await (await MasterChef.transferOwnership(
      deployAddress, 
      gasSettings
    )).wait()
  }

  const UniswapV2Factory = await deploy({
    contractName: "UniswapV2Factory",
    rpcUrl: l2RpcUrl,
    pk: deployPrivateKey,
    ovm: true,
    _args: [deployAddress],
  })

  const UniswapV2Router02 = await deploy({
    contractName: "UniswapV2Router02",
    rpcUrl: l2RpcUrl,
    pk: deployPrivateKey,
    ovm: true,
    _args: [UniswapV2Factory.address, l2ETHAddress],
  })

  //const UNISWAP_ROUTER = new Map()
  //UNISWAP_ROUTER.set("1", "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D")

  const SushiRoll = await deploy({
    contractName: "SushiRoll",
    rpcUrl: l2RpcUrl,
    pk: deployPrivateKey,
    ovm: true,
    // _args: ["0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D", UniswapV2Router02.address]
    _args: [l2ETHAddress,l2ETHAddress]
  })

  // save address
  const addresses = {
    SushiToken: SushiToken.address,
    SushiBar: SushiBar.address,
    MasterChef: MasterChef.address,
    UniswapV2Factory: UniswapV2Factory.address,
    UniswapV2Router02: UniswapV2Router02.address,
    SushiRoll: SushiRoll.address,
  }

  writeFileSyncRecursive(`/deployments/addresses.json`,JSON.stringify(addresses));

  console.log(
    "\n\n 🛰  Addresses: \n",
    addresses,
  )
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });

exports.deploy = deploy;
