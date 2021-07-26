import {World} from './World';
import {readFile} from './File';
import request from 'request';
import * as path from 'path';
import truffleFlattener from 'truffle-flattener';
import {getNetworkContracts} from './Contract';

interface DevDoc {
  author: string
  methods: object
  title: string
}

interface UserDoc {
  methods: object
  notice: string
}

function getUrl(network: string): string {
  let host = {
    kovan: 'api-kovan.etherscan.io',
    rinkeby: 'api-rinkeby.etherscan.io',
    ropsten: 'api-ropsten.etherscan.io',
    goerli: 'api-goerli.etherscan.io',
    mainnet: 'api.etherscan.io'
  }[network];

  if (!host) {
    throw new Error(`Unknown etherscan API host for network ${network}`);
  }

  return `https://${host}/api`;
}

function getConstructorABI(world: World, contractName: string): string {
  let constructorAbi = world.getIn(['contractData', 'Constructors', contractName]);

  if (!constructorAbi) {
    throw new Error(`Unknown Constructor ABI for ${contractName} on ${world.network}. Try deploying again?`);
  }

  return constructorAbi;
}

function post(url, data): Promise<object> {
  return new Promise((resolve, reject) => {
    request.post(url, {form: data}, (err, httpResponse, body) => {
      if (err) {
        reject(err);
      } else {
        resolve(JSON.parse(body));
      }
    });
  });
}

function get(url, data): Promise<object> {
  return new Promise((resolve, reject) => {
    request.get(url, {form: data}, (err, httpResponse, body) => {
      if (err) {
        reject(err);
      } else {
        resolve(JSON.parse(body));
      }
    });
  });
}

interface Result {
  status: string
  message: string
  result: string
}

async function sleep(timeout): Promise<void> {
  return new Promise((resolve, _reject) => {
    setTimeout(() => resolve(), timeout);
  })
}

async function checkStatus(world: World, url: string, token: string): Promise<void> {
  world.printer.printLine(`Checking status of ${token}...`);

  // Potential results:
  // { status: '0', message: 'NOTOK', result: 'Fail - Unable to verify' }
  // { status: '0', message: 'NOTOK', result: 'Pending in queue' }
  // { status: '1', message: 'OK', result: 'Pass - Verified' }

  let result: Result = <Result>await get(url, {
    guid: token,
    module: "contract",
    action: "checkverifystatus"
  });

  if (world.verbose) {
    console.log(result);
  }

  if (result.result === "Pending in queue") {
    await sleep(5000);
    return await checkStatus(world, url, token);
  }

  if (result.result.startsWith('Fail')) {
    throw new Error(`Etherscan failed to verify contract: ${result.message} "${result.result}"`)
  }

  if (Number(result.status) !== 1) {
    throw new Error(`Etherscan Error: ${result.message} "${result.result}"`)
  }

  world.printer.printLine(`Verification result ${result.result}...`);
}

export async function verify(world: World, apiKey: string, contractName: string, buildInfoName: string, address: string): Promise<void> {
  let contractAddress: string = address;
  let {networkContracts, version} = await getNetworkContracts(world);
  let networkContract = networkContracts[buildInfoName];
  if (!networkContract) {
    throw new Error(`Cannot find contract ${buildInfoName}, found: ${Object.keys(networkContracts)}`)
  }
  let sourceCode: string = await truffleFlattener([networkContract.path]);
  let compilerVersion: string = version.replace(/(\.Emscripten)|(\.clang)|(\.Darwin)|(\.appleclang)/gi, '');
  let constructorAbi = getConstructorABI(world, contractName);
  let url = getUrl(world.network);

  const verifyData: object = {
    apikey: apiKey,
    module: 'contract',
    action: 'verifysourcecode',
    contractaddress: contractAddress,
    sourceCode: sourceCode,
    contractname: buildInfoName,
    compilerversion: `v${compilerVersion}`,
    optimizationUsed: '1',
    runs: '200',
    constructorArguements: constructorAbi.slice(2)
  };

  world.printer.printLine(`Verifying ${contractName} at ${address} with compiler version ${compilerVersion}...`);

  // Potential results
  // {"status":"0","message":"NOTOK","result":"Invalid constructor arguments provided. Please verify that they are in ABI-encoded format"}
  // {"status":"1","message":"OK","result":"usjpiyvmxtgwyee59wnycyiet7m3dba4ccdi6acdp8eddlzdde"}

  let result: Result = <Result>await post(url, verifyData);

  if (Number(result.status) === 0 || result.message !== "OK") {
    if (result.result.includes('Contract source code already verified')) {
      world.printer.printLine(`Contract already verified`);
    } else {
      throw new Error(`Etherscan Error: ${result.message}: ${result.result}`)
    }
  } else {
    return await checkStatus(world, url, result.result);
  }
}
