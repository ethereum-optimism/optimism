import {ReplPrinter} from './Printer';
import {
  addInvariant,
  initWorld,
  loadInvokationOpts,
  loadDryRun,
  loadSettings,
  loadVerbose,
  World
} from './World';
import {throwExpect} from './Assert';
import {Macros} from './Macro';
import {formatEvent} from './Formatter';
import {complete} from './Completer';
import {loadContracts} from './Networks';
import {accountAliases, loadAccounts} from './Accounts';
import {getNetworkPath, readFile} from './File';
import {SuccessInvariant} from './Invariant/SuccessInvariant';
import {createInterface} from './HistoricReadline';
import {runCommand} from './Runner';
import {parse} from './Parser';
import {forkWeb3} from './Hypothetical';
import {getSaddle} from 'eth-saddle';
import Web3 from 'web3';

import * as fs from 'fs';
import * as path from 'path';

const basePath = process.env.proj_root || process.cwd();
const baseScenarioPath = path.join(basePath, 'spec', 'scenario');
const baseNetworksPath = path.join(basePath, 'networks');

const TOTAL_GAS = 8000000;

function questionPromise(rl): Promise<string> {
  return new Promise((resolve, reject) => {
    rl.question(" > ", (command) => {
      resolve(command);
    });
  });
}

async function loop(world, rl, macros): Promise<any> {
  let command = await questionPromise(rl);

  try {
    let newWorld = await runCommand(world, command, macros);

    return await loop(newWorld, rl, macros);
  } catch (err) {
    world.printer.printError(err);
    return await loop(world, rl, macros);
  }
}

function loadEnvVars(): object {
  return (process.env['env_vars'] || '').split(',').reduce((acc, keyValue) => {
    if (keyValue.length === 0) {
      return acc;
    } else {
      const [key, value] = keyValue.split('=');

      return {
        ...acc,
        [key]: value
      };
    }
  }, {});
}

async function repl(): Promise<void> {
  // Uck, we need to load core macros :(
  const coreMacros = fs.readFileSync(path.join(baseScenarioPath, 'CoreMacros'), 'utf8');

  const macros = <Macros>parse(coreMacros, {startRule: 'macros'});

  let script = process.env['script'];

  let network = process.env['network'];

  if (!network) {
    throw new Error(`Missing required "network" env argument`);
  }

  let world;

  let rl = await createInterface({
    input: process.stdin,
    output: process.stdout,
    completer: (line) => complete(world, macros, line),
    path: getNetworkPath(basePath, network, '-history', null)
  });

  const verbose: boolean = !!process.env['verbose'];
  const hypothetical: boolean = !!process.env['hypothetical'];

  let printer = new ReplPrinter(rl, verbose);
  let contractInfo: string[];

  let saddle = await getSaddle(network);
  let accounts: string[] = saddle.wallet_accounts.concat(saddle.accounts).filter((x) => !!x);

  world = await initWorld(throwExpect, printer, saddle.web3, saddle, network, accounts, basePath, TOTAL_GAS);
  [world, contractInfo] = await loadContracts(world);
  world = loadInvokationOpts(world);
  world = loadVerbose(world);
  world = loadDryRun(world);
  world = await loadSettings(world);

  printer.printLine(`Network: ${network}`);

  if (hypothetical) {
    const forkJsonPath = path.join(baseNetworksPath, `${network}-fork.json`);
    let forkJson;

    try {
      let forkJsonString = fs.readFileSync(forkJsonPath, 'utf8');
      forkJson = JSON.parse(forkJsonString);
    } catch (err) {
      throw new Error(`Cannot read fork configuration from \`${forkJsonPath}\`, ${err}`);
    }
    if (!forkJson['url']) {
      throw new Error(`Missing url in fork json`);
    }
    if (!forkJson['unlocked'] || !Array.isArray(forkJson.unlocked)) {
      throw new Error(`Missing unlocked in fork json`);
    }

    saddle.web3 = await forkWeb3(saddle.web3, forkJson.url, forkJson.unlocked);
    saddle.accounts = forkJson.unlocked;
    console.log(`Running on fork ${forkJson.url} with unlocked accounts ${forkJson.unlocked.join(', ')}`)
  }

  if (accounts.length > 0) {
    printer.printLine(`Accounts:`);
    accounts.forEach((account, i) => {
      let aliases = world.settings.lookupAliases(account);
      aliases = aliases.concat(accountAliases(i));

      printer.printLine(`\t${account} (${aliases.join(',')})`)
    });
  }

  if (contractInfo.length > 0) {
    world.printer.printLine(`Contracts:`);
    contractInfo.forEach((info) => world.printer.printLine(`\t${info}`));
  }

  printer.printLine(`Available macros: ${Object.keys(macros).toString()}`);
  printer.printLine(``);

  if (script) {
    const combined = script.split(',').reduce((acc, script) => {
      printer.printLine(`Running script: ${script}...`);
      const envVars = loadEnvVars();
      if (hypothetical) {
        envVars['hypo'] = true;
      }
      const scriptData: string = fs.readFileSync(script).toString();

      if (Object.keys(envVars).length > 0) {
        printer.printLine(`Env Vars:`);
      }

      const replacedScript = Object.entries(envVars).reduce((data, [key, val]) => {
        printer.printLine(`\t${key}: ${val}`);

        return data.split(`$${key}`).join(val);
      }, scriptData);

      const finalScript = replacedScript.replace(new RegExp(/\$[\w_]+/, 'g'), 'Nothing');

      return [...acc, ...finalScript.split("\n")];
    }, <string[]>[]);

    return await combined.reduce(async (acc, command) => {
      return await runCommand(await acc, command, macros);
    }, Promise.resolve(world));
    printer.printLine(`Script complete.`);
  } else {
    await loop(world, rl, macros);
  }
}

repl().catch((error) => {
  console.error(error);
  process.exit(1);
});
