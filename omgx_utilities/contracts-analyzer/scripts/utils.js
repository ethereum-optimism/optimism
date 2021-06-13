/* eslint no-use-before-define: "warn" */
const fs = require("fs");
const chalk = require("chalk");
const { l2ethers } = require("hardhat");
const { utils } = require("ethers");
const R = require("ramda");

async function deploy({
  rpcUrl, 
  contractName, 
  pk, ovm = false, 
  _args = [], 
  overrides = {}, 
}) {
  
  console.log(` 🛰  ${ovm?`OVM`:`EVM`} Deploying: ${contractName} on ${rpcUrl}`);

  const contractArgs = _args || [];

  const provider = new ethers.providers.JsonRpcProvider(rpcUrl)
  const signerProvider = new ethers.Wallet(pk).connect(provider)

  let contractArtifacts

  if(ovm === true) {
    contractArtifacts = await l2ethers.getContractFactory(contractName, signerProvider);
  } else {
    contractArtifacts = await ethers.getContractFactory(contractName, signerProvider);
  }
  
  const nonce = await signerProvider.getTransactionCount()
  const deployed = await contractArtifacts.deploy(...contractArgs, { nonce, ...overrides, gasPrice: 0, gasLimit: 800000 });
  await deployed.deployTransaction.wait()

  const checkCode = async (_address) => {
    let code = await provider.getCode(_address)
    console.log(` 📱 Code: ${code.slice(0, 100)}`)
  }

  let result = await checkCode(deployed.address)
  if(result=="0x"){
    console.log("☢️☢️☢️☢️☢️ CONTRACT DID NOT DEPLOY ☢️☢️☢️☢️☢️")
    return 0
  }

  const encoded = abiEncodeArgs(deployed, contractArgs);
  fs.writeFileSync(`artifacts-ovm/${contractName}.address`, deployed.address);

  let extraGasInfo = ""
  if(deployed&&deployed.deployTransaction){
    const gasUsed = deployed.deployTransaction.gasLimit.mul(deployed.deployTransaction.gasPrice)
    extraGasInfo = "("+utils.formatEther(gasUsed)+" ETH)"
  }

  console.log(
    " 📄",
    chalk.cyan(contractName),
    "deployed to:",
    chalk.magenta(deployed.address),
    chalk.grey(extraGasInfo)
  );

  if (!encoded || encoded.length <= 2) return deployed;
  fs.writeFileSync(`artifacts-ovm/${contractName}.args`, encoded.slice(2));

  return deployed;
};


// abi encodes contract arguments
// useful when you want to manually verify the contracts
// for example, on Etherscan
const abiEncodeArgs = (deployed, contractArgs) => {
  // not writing abi encoded args if this does not pass
  if (
    !contractArgs ||
    !deployed ||
    !R.hasPath(["interface", "deploy"], deployed)
  ) {
    return "";
  }
  const encoded = utils.defaultAbiCoder.encode(
    deployed.interface.deploy.inputs,
    contractArgs
  );
  return encoded;
};

exports.deploy = deploy;
