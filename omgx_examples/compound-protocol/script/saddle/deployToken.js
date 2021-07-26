let { loadConf } = require('./support/tokenConfig');

function printUsage() {
  console.log(`
usage: npx saddle script token:deploy {tokenConfig}

note: pass VERIFY=true and ETHERSCAN_API_KEY=<api key> to verify contract on Etherscan

example:

npx saddle -n rinkeby script token:deploy '{
  "underlying": "0x577D296678535e4903D59A4C929B718e1D575e0A",
  "comptroller": "$Comptroller",
  "interestRateModel": "$Base200bps_Slope3000bps",
  "initialExchangeRateMantissa": "2.0e18",
  "name": "Compound Kyber Network Crystal",
  "symbol": "cKNC",
  "decimals": "8",
  "admin": "$Timelock"
}'
  `);
}

function sleep(timeout) {
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      resolve();
    }, timeout);
  });
}

(async function() {
  if (args.length !== 1) {
    return printUsage();
  }

  let conf = loadConf(args[0], addresses);
  if (!conf) {
    return printUsage();
  }

  console.log(`Deploying cToken with ${JSON.stringify(conf)}`);

  let deployArgs = [conf.underlying, conf.comptroller, conf.interestRateModel, conf.initialExchangeRateMantissa.toString(), conf.name, conf.symbol, conf.decimals, conf.admin];
  let contract = await saddle.deploy('CErc20Immutable', deployArgs);

  console.log(`Deployed contract to ${contract._address}`);

  if (env['VERIFY']) {
    const etherscanApiKey = env['ETHERSCAN_API_KEY'];
    if (etherscanApiKey === undefined || etherscanApiKey.length === 0) {
      throw new Error(`ETHERSCAN_API_KEY must be set if using VERIFY flag...`);
    }

    console.log(`Sleeping for 30 seconds then verifying contract on Etherscan...`);
    await sleep(30000); // Give Etherscan time to learn about contract
    console.log(`Now verifying contract on Etherscan...`);

    await saddle.verify(etherscanApiKey, contract._address, 'CErc20Immutable', deployArgs, 0);
    console.log(`Contract verified at https://${network}.etherscan.io/address/${contract._address}`);
  }

  return {
    ...conf,
    address: contract._address
  };
})();
