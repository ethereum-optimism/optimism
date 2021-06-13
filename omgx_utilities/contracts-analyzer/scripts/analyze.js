const fs = require("fs");
const chalk = require("chalk");
const bre = require("hardhat");

async function main() {
  
  console.log(" 📡 Analyzing...\n");

  try {
    
    const alertMessages = { critical: 0, minor: 0 }; 
    const dirents = fs.readdirSync(`contracts`, { withFileTypes: true });
    
    const contractNames = dirents
      .filter(dirent => dirent.isFile())
      .filter(dirent => !(dirent.name === '.DS_Store'))
      .map(dirent => dirent.name.replace(".sol", ""));
    
    const inlineAssemblyReg = /\bassembly([\s]*)\{{1}?\s?(.)?[\s\w:=(),]+\}/gm;
    
    for (let contractName of contractNames) {
      console.log(` 📡 Analyzing ${chalk.cyan(contractName)}`);
      
      const contract = fs
        .readFileSync(`${bre.config.paths.artifacts}-ovm/contracts/${contractName}.sol/${contractName}.json`)
        .toString();

      const sizeByte = Buffer.from(
        JSON.parse(contract).deployedBytecode.replace(/__\$\w*\$__/g, '0'.repeat(40)).slice(2),
        'hex'
      ).length;
      
      const sizeKB = (sizeByte / 1024).toFixed(2);

      console.log(` 💡${chalk.cyan(contractName)} size: ${sizeKB} kB`);
      
      if (sizeKB > 18) {
        console.log(` 🚨${chalk.red(`${chalk.cyan(contractName)} is larger than 18 kB. It might fail to deploy on L2. Please consider reducing the contract's size.`)}`);
        alertMessages.minor += 1;
      }
      
      const contractSource = fs
        .readFileSync(`${bre.config.paths.sources}/${contractName}.sol`)
        .toString();

      const contractSourceArr = contractSource.split("\n");
      while ((result = inlineAssemblyReg.exec(contractSource)) !== null) {
        let index = 0;
        const startIndex = result.index;
        for (let i in contractSourceArr) {
          if (index <= startIndex && index + contractSourceArr[i].length - 1 >= startIndex) {
            console.log(` 🚨${chalk.red(`Found inline Assembly at ${bre.config.paths.sources}/${contractName}.sol#L${Number(i) + 1}`)}`);
            alertMessages.critical += 1;
            break;
          }
          index += contractSourceArr[i].length + 1;
        }
      }

      console.log('\n');
    }

    if (alertMessages.critical === 0) {
      console.log(` 🌕Passed\n`);
    } else {
      console.log(` 🚨${chalk.red(`Found ${alertMessages.critical} critical issues. Please fix them before deploying contracts.`)}\n`);
    }

    if (alertMessages.minor === 1) {
      console.log(` 🚨${chalk.red(`Found 1 minor issue. Please review it before deploying contracts.`)}\n`);
    } else if (alertMessages.minor > 1) {
      console.log(` 🚨${chalk.red(`Found ${alertMessages.minor} minor issues. Please review them before deploying contracts.`)}\n`);
    }
    
  } catch(e) {
    if(e.toString().indexOf("no such file or directory")>=0){
      console.log(e)
      return false
    }else{
      console.log(e);
      return false;
    }
  }
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });