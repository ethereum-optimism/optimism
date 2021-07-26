
function getRaw(config, key, required=true) {
  let value = config[key];
  if (required && !value) {
    throw new Error(`Config missing required key \`${key}\``);
  }
  return value;
}

function getString(config, key, required=true) {
  let value = getRaw(config, key, required);
  if (value === "" && required) {
    throw new Error(`Config missing required key \`${key}\``);
  }
  return value || "";
}

function loadAddress(value, addresses, errorMessage=null) {
  if (value.startsWith("$")) {
    let contract = value.slice(1,);
    let address = addresses[contract];
    if (!address) {
      throw new Error(`Cannot find deploy address for \`${contract}\``);
    }
    return address;
  } else if (value.startsWith("0x")) {
    return value;
  } else {
    throw new Error(errorMessage || `Invalid address \`${value}\``);
  }
}

function getAddress(addresses, config, key, required=true) {
  let value = getString(config, key, required);
  return loadAddress(
    value,
    addresses,
    `Invalid address for \`${key}\`=${value}`,
    required
  );
}

function getNumber(config, key, required=true) {
  let value = getRaw(config, key, required);
  let result = Number(value);
  if (value == null && !required){
    return null;
  } else if (Number.isNaN(result)) {
    throw new Error(`Invalid number for \`${key}\`=${value}`);
  } else {
    return result;
  }
}

function getArray(config, key, required = true) {
  let value = getRaw(config, key, required);
  if (value == null && !required){
    return null;
  } else if (Array.isArray(value)) {
    return value;
  } else {
    throw new Error(`Invalid array for \`${key}\`=${value}`);
  }
}

function getBoolean(config, key, required = true) {
  let value = getRaw(config, key, required);
  if (value == null && !required){
    return null;
  } else if (value === "false" || value === "true") {
    return value == 'true';
  } else {
    throw new Error(`Invalid bool for \`${key}\`=${value}`);
  }
}

function loadConf(configArg, addresses) {
  let config;
  if (!configArg) {
    return null;
  }

  try {
    config = JSON.parse(configArg)
  } catch (e) {
    console.log();
    console.error(e);
    return null;
  }
  const conf = {
    underlying: getAddress(addresses, config, 'underlying'),
    comptroller: getAddress(addresses, config, 'comptroller'),
    interestRateModel: getAddress(addresses, config, 'interestRateModel'),
    initialExchangeRateMantissa: getNumber(config, 'initialExchangeRateMantissa'),
    name: getString(config, 'name'),
    symbol: getString(config, 'symbol'),
    decimals: getNumber(config, 'decimals'),
    admin: getAddress(addresses, config, 'admin'),
  };

  return conf;
}

module.exports = {
  loadAddress,
  loadConf,
  getNumber,
  getArray,
  getBoolean
};
