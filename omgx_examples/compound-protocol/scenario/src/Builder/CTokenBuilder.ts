import { Event } from '../Event';
import { World } from '../World';
import { CErc20Delegator, CErc20DelegatorScenario } from '../Contract/CErc20Delegator';
import { CToken } from '../Contract/CToken';
import { Invokation, invoke } from '../Invokation';
import { getAddressV, getExpNumberV, getNumberV, getStringV } from '../CoreValue';
import { AddressV, NumberV, StringV } from '../Value';
import { Arg, Fetcher, getFetcherValue } from '../Command';
import { storeAndSaveContract } from '../Networks';
import { getContract, getTestContract } from '../Contract';

const CErc20Contract = getContract('CErc20Immutable');
const CErc20Delegator = getContract('CErc20Delegator');
const CErc20DelegatorScenario = getTestContract('CErc20DelegatorScenario');
const CEtherContract = getContract('CEther');
const CErc20ScenarioContract = getTestContract('CErc20Scenario');
const CEtherScenarioContract = getTestContract('CEtherScenario');
const CEvilContract = getTestContract('CEvil');

export interface TokenData {
  invokation: Invokation<CToken>;
  name: string;
  symbol: string;
  decimals?: number;
  underlying?: string;
  address?: string;
  contract: string;
  initial_exchange_rate_mantissa?: string;
  admin?: string;
}

export async function buildCToken(
  world: World,
  from: string,
  params: Event
): Promise<{ world: World; cToken: CToken; tokenData: TokenData }> {
  const fetchers = [
    new Fetcher<
      {
        symbol: StringV;
        name: StringV;
        decimals: NumberV;
        underlying: AddressV;
        comptroller: AddressV;
        interestRateModel: AddressV;
        initialExchangeRate: NumberV;
        admin: AddressV;
        implementation: AddressV;
        becomeImplementationData: StringV;
      },
      TokenData
    >(
    `
      #### CErc20Delegator

      * "CErc20Delegator symbol:<String> name:<String> underlying:<Address> comptroller:<Address> interestRateModel:<Address> initialExchangeRate:<Number> decimals:<Number> admin: <Address> implementation:<Address> becomeImplementationData:<String>" - The real deal CToken
        * E.g. "CToken Deploy CErc20Delegator cDAI \"Compound DAI\" (Erc20 DAI Address) (Comptroller Address) (InterestRateModel Address) 1.0 8 Geoff (CToken CDaiDelegate Address) "0x0123434anyByTes314535q" "
    `,
      'CErc20Delegator',
      [
        new Arg('symbol', getStringV),
        new Arg('name', getStringV),
        new Arg('underlying', getAddressV),
        new Arg('comptroller', getAddressV),
        new Arg('interestRateModel', getAddressV),
        new Arg('initialExchangeRate', getExpNumberV),
        new Arg('decimals', getNumberV),
        new Arg('admin', getAddressV),
        new Arg('implementation', getAddressV),
        new Arg('becomeImplementationData', getStringV)
      ],
      async (
        world,
        {
          symbol,
          name,
          underlying,
          comptroller,
          interestRateModel,
          initialExchangeRate,
          decimals,
          admin,
          implementation,
          becomeImplementationData
        }
      ) => {
        return {
          invokation: await CErc20Delegator.deploy<CErc20Delegator>(world, from, [
            underlying.val,
            comptroller.val,
            interestRateModel.val,
            initialExchangeRate.val,
            name.val,
            symbol.val,
            decimals.val,
            admin.val,
            implementation.val,
            becomeImplementationData.val
          ]),
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          underlying: underlying.val,
          contract: 'CErc20Delegator',
          initial_exchange_rate_mantissa: initialExchangeRate.encode().toString(),
          admin: admin.val
        };
      }
    ),

    new Fetcher<
      {
        symbol: StringV;
        name: StringV;
        decimals: NumberV;
        underlying: AddressV;
        comptroller: AddressV;
        interestRateModel: AddressV;
        initialExchangeRate: NumberV;
        admin: AddressV;
        implementation: AddressV;
        becomeImplementationData: StringV;
      },
      TokenData
    >(
    `
      #### CErc20DelegatorScenario

      * "CErc20DelegatorScenario symbol:<String> name:<String> underlying:<Address> comptroller:<Address> interestRateModel:<Address> initialExchangeRate:<Number> decimals:<Number> admin: <Address> implementation:<Address> becomeImplementationData:<String>" - A CToken Scenario for local testing
        * E.g. "CToken Deploy CErc20DelegatorScenario cDAI \"Compound DAI\" (Erc20 DAI Address) (Comptroller Address) (InterestRateModel Address) 1.0 8 Geoff (CToken CDaiDelegate Address) "0x0123434anyByTes314535q" "
    `,
      'CErc20DelegatorScenario',
      [
        new Arg('symbol', getStringV),
        new Arg('name', getStringV),
        new Arg('underlying', getAddressV),
        new Arg('comptroller', getAddressV),
        new Arg('interestRateModel', getAddressV),
        new Arg('initialExchangeRate', getExpNumberV),
        new Arg('decimals', getNumberV),
        new Arg('admin', getAddressV),
        new Arg('implementation', getAddressV),
        new Arg('becomeImplementationData', getStringV)
      ],
      async (
        world,
        {
          symbol,
          name,
          underlying,
          comptroller,
          interestRateModel,
          initialExchangeRate,
          decimals,
          admin,
          implementation,
          becomeImplementationData
        }
      ) => {
        return {
          invokation: await CErc20DelegatorScenario.deploy<CErc20DelegatorScenario>(world, from, [
            underlying.val,
            comptroller.val,
            interestRateModel.val,
            initialExchangeRate.val,
            name.val,
            symbol.val,
            decimals.val,
            admin.val,
            implementation.val,
            becomeImplementationData.val
          ]),
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          underlying: underlying.val,
          contract: 'CErc20DelegatorScenario',
          initial_exchange_rate_mantissa: initialExchangeRate.encode().toString(),
          admin: admin.val
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV, underlying: AddressV, comptroller: AddressV, interestRateModel: AddressV, initialExchangeRate: NumberV, admin: AddressV}, TokenData>(`
        #### Scenario

        * "Scenario symbol:<String> name:<String> underlying:<Address> comptroller:<Address> interestRateModel:<Address> initialExchangeRate:<Number> decimals:<Number> admin: <Address>" - A CToken Scenario for local testing
          * E.g. "CToken Deploy Scenario cZRX \"Compound ZRX\" (Erc20 ZRX Address) (Comptroller Address) (InterestRateModel Address) 1.0 8"
      `,
      "Scenario",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("underlying", getAddressV),
        new Arg("comptroller", getAddressV),
        new Arg("interestRateModel", getAddressV),
        new Arg("initialExchangeRate", getExpNumberV),
        new Arg("decimals", getNumberV),
        new Arg("admin", getAddressV)
      ],
      async (world, {symbol, name, underlying, comptroller, interestRateModel, initialExchangeRate, decimals, admin}) => {
        return {
          invokation: await CErc20ScenarioContract.deploy<CToken>(world, from, [underlying.val, comptroller.val, interestRateModel.val, initialExchangeRate.val, name.val, symbol.val, decimals.val, admin.val]),
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          underlying: underlying.val,
          contract: 'CErc20Scenario',
          initial_exchange_rate_mantissa: initialExchangeRate.encode().toString(),
          admin: admin.val
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV, admin: AddressV, comptroller: AddressV, interestRateModel: AddressV, initialExchangeRate: NumberV}, TokenData>(`
        #### CEtherScenario

        * "CEtherScenario symbol:<String> name:<String> comptroller:<Address> interestRateModel:<Address> initialExchangeRate:<Number> decimals:<Number> admin: <Address>" - A CToken Scenario for local testing
          * E.g. "CToken Deploy CEtherScenario cETH \"Compound Ether\" (Comptroller Address) (InterestRateModel Address) 1.0 8"
      `,
      "CEtherScenario",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("comptroller", getAddressV),
        new Arg("interestRateModel", getAddressV),
        new Arg("initialExchangeRate", getExpNumberV),
        new Arg("decimals", getNumberV),
        new Arg("admin", getAddressV)
      ],
      async (world, {symbol, name, comptroller, interestRateModel, initialExchangeRate, decimals, admin}) => {
        return {
          invokation: await CEtherScenarioContract.deploy<CToken>(world, from, [name.val, symbol.val, decimals.val, admin.val, comptroller.val, interestRateModel.val, initialExchangeRate.val]),
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          underlying: "",
          contract: 'CEtherScenario',
          initial_exchange_rate_mantissa: initialExchangeRate.encode().toString(),
          admin: admin.val
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV, admin: AddressV, comptroller: AddressV, interestRateModel: AddressV, initialExchangeRate: NumberV}, TokenData>(`
        #### CEther

        * "CEther symbol:<String> name:<String> comptroller:<Address> interestRateModel:<Address> initialExchangeRate:<Number> decimals:<Number> admin: <Address>" - A CToken Scenario for local testing
          * E.g. "CToken Deploy CEther cETH \"Compound Ether\" (Comptroller Address) (InterestRateModel Address) 1.0 8"
      `,
      "CEther",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("comptroller", getAddressV),
        new Arg("interestRateModel", getAddressV),
        new Arg("initialExchangeRate", getExpNumberV),
        new Arg("decimals", getNumberV),
        new Arg("admin", getAddressV)
      ],
      async (world, {symbol, name, comptroller, interestRateModel, initialExchangeRate, decimals, admin}) => {
        return {
          invokation: await CEtherContract.deploy<CToken>(world, from, [comptroller.val, interestRateModel.val, initialExchangeRate.val, name.val, symbol.val, decimals.val, admin.val]),
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          underlying: "",
          contract: 'CEther',
          initial_exchange_rate_mantissa: initialExchangeRate.encode().toString(),
          admin: admin.val
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV, admin: AddressV, underlying: AddressV, comptroller: AddressV, interestRateModel: AddressV, initialExchangeRate: NumberV}, TokenData>(`
        #### CErc20

        * "CErc20 symbol:<String> name:<String> underlying:<Address> comptroller:<Address> interestRateModel:<Address> initialExchangeRate:<Number> decimals:<Number> admin: <Address>" - A official CToken contract
          * E.g. "CToken Deploy CErc20 cZRX \"Compound ZRX\" (Erc20 ZRX Address) (Comptroller Address) (InterestRateModel Address) 1.0 8"
      `,
      "CErc20",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("underlying", getAddressV),
        new Arg("comptroller", getAddressV),
        new Arg("interestRateModel", getAddressV),
        new Arg("initialExchangeRate", getExpNumberV),
        new Arg("decimals", getNumberV),
        new Arg("admin", getAddressV)
      ],
      async (world, {symbol, name, underlying, comptroller, interestRateModel, initialExchangeRate, decimals, admin}) => {

        return {
          invokation: await CErc20Contract.deploy<CToken>(world, from, [underlying.val, comptroller.val, interestRateModel.val, initialExchangeRate.val, name.val, symbol.val, decimals.val, admin.val]),
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          underlying: underlying.val,
          contract: 'CErc20',
          initial_exchange_rate_mantissa: initialExchangeRate.encode().toString(),
          admin: admin.val
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV, admin: AddressV, underlying: AddressV, comptroller: AddressV, interestRateModel: AddressV, initialExchangeRate: NumberV}, TokenData>(`
        #### CEvil

        * "CEvil symbol:<String> name:<String> underlying:<Address> comptroller:<Address> interestRateModel:<Address> initialExchangeRate:<Number> decimals:<Number> admin: <Address>" - A malicious CToken contract
          * E.g. "CToken Deploy CEvil cEVL \"Compound EVL\" (Erc20 ZRX Address) (Comptroller Address) (InterestRateModel Address) 1.0 8"
      `,
      "CEvil",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("underlying", getAddressV),
        new Arg("comptroller", getAddressV),
        new Arg("interestRateModel", getAddressV),
        new Arg("initialExchangeRate", getExpNumberV),
        new Arg("decimals", getNumberV),
        new Arg("admin", getAddressV)
      ],
      async (world, {symbol, name, underlying, comptroller, interestRateModel, initialExchangeRate, decimals, admin}) => {
        return {
          invokation: await CEvilContract.deploy<CToken>(world, from, [underlying.val, comptroller.val, interestRateModel.val, initialExchangeRate.val, name.val, symbol.val, decimals.val, admin.val]),
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          underlying: underlying.val,
          contract: 'CEvil',
          initial_exchange_rate_mantissa: initialExchangeRate.encode().toString(),
          admin: admin.val
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV, admin: AddressV, underlying: AddressV, comptroller: AddressV, interestRateModel: AddressV, initialExchangeRate: NumberV}, TokenData>(`
        #### Standard

        * "symbol:<String> name:<String> underlying:<Address> comptroller:<Address> interestRateModel:<Address> initialExchangeRate:<Number> decimals:<Number> admin: <Address>" - A official CToken contract
          * E.g. "CToken Deploy Standard cZRX \"Compound ZRX\" (Erc20 ZRX Address) (Comptroller Address) (InterestRateModel Address) 1.0 8"
      `,
      "Standard",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("underlying", getAddressV),
        new Arg("comptroller", getAddressV),
        new Arg("interestRateModel", getAddressV),
        new Arg("initialExchangeRate", getExpNumberV),
        new Arg("decimals", getNumberV),
        new Arg("admin", getAddressV)
      ],
      async (world, {symbol, name, underlying, comptroller, interestRateModel, initialExchangeRate, decimals, admin}) => {
        // Note: we're going to use the scenario contract as the standard deployment on local networks
        if (world.isLocalNetwork()) {
          return {
            invokation: await CErc20ScenarioContract.deploy<CToken>(world, from, [underlying.val, comptroller.val, interestRateModel.val, initialExchangeRate.val, name.val, symbol.val, decimals.val, admin.val]),
            name: name.val,
            symbol: symbol.val,
            decimals: decimals.toNumber(),
            underlying: underlying.val,
            contract: 'CErc20Scenario',
            initial_exchange_rate_mantissa: initialExchangeRate.encode().toString(),
            admin: admin.val
          };
        } else {
          return {
            invokation: await CErc20Contract.deploy<CToken>(world, from, [underlying.val, comptroller.val, interestRateModel.val, initialExchangeRate.val, name.val, symbol.val, decimals.val, admin.val]),
            name: name.val,
            symbol: symbol.val,
            decimals: decimals.toNumber(),
            underlying: underlying.val,
            contract: 'CErc20Immutable',
            initial_exchange_rate_mantissa: initialExchangeRate.encode().toString(),
            admin: admin.val
          };
        }
      },
      {catchall: true}
    )
  ];

  let tokenData = await getFetcherValue<any, TokenData>("DeployCToken", fetchers, world, params);
  let invokation = tokenData.invokation;
  delete tokenData.invokation;

  if (invokation.error) {
    throw invokation.error;
  }

  const cToken = invokation.value!;
  tokenData.address = cToken._address;

  world = await storeAndSaveContract(
    world,
    cToken,
    tokenData.symbol,
    invokation,
    [
      { index: ['cTokens', tokenData.symbol], data: tokenData },
      { index: ['Tokens', tokenData.symbol], data: tokenData }
    ]
  );

  return {world, cToken, tokenData};
}
