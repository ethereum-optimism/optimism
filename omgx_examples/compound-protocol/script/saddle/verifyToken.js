let { loadAddress, loadConf } = require('./support/tokenConfig');

function printUsage() {
  console.log(`
usage: npx saddle script token:verify {tokenAddress} {tokenConfig}

note: $ETHERSCAN_API_KEY environment variable must be set to an Etherscan API Key.

example:

npx saddle -n rinkeby script token:verify 0x19B674715cD20626415C738400FDd0d32D6809B6 '{
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

(async function() {
  if (args.length !== 2) {
    return printUsage();
  }

  let address = loadAddress(args[0], addresses);
  let conf = loadConf(args[1], addresses);
  if (!conf) {
    return printUsage();
  }
  let etherscanApiKey = env['ETHERSCAN_API_KEY'];
  if (!etherscanApiKey) {
    console.error("Missing required $ETHERSCAN_API_KEY env variable.");
    return printUsage();
  }

  console.log(`Verifying cToken at ${address} with ${JSON.stringify(conf)}`);

  let deployArgs = [conf.underlying, conf.comptroller, conf.interestRateModel, conf.initialExchangeRateMantissa.toString(), conf.name, conf.symbol, conf.decimals, conf.admin];

  // TODO: Make sure we match optimizations count, etc
  await saddle.verify(etherscanApiKey, address, 'CErc20Immutable', deployArgs, 200, undefined);

  console.log(`Contract verified at https://${network}.etherscan.io/address/${address}`);

  return {
    ...conf,
    address
  };
})();
