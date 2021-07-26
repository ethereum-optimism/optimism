import { Event } from '../Event';
import { World, addAction } from '../World';
import { Comp, CompScenario } from '../Contract/Comp';
import { Invokation } from '../Invokation';
import { getAddressV } from '../CoreValue';
import { StringV, AddressV } from '../Value';
import { Arg, Fetcher, getFetcherValue } from '../Command';
import { storeAndSaveContract } from '../Networks';
import { getContract } from '../Contract';

const CompContract = getContract('Comp');
const CompScenarioContract = getContract('CompScenario');

export interface TokenData {
  invokation: Invokation<Comp>;
  contract: string;
  address?: string;
  symbol: string;
  name: string;
  decimals?: number;
}

export async function buildComp(
  world: World,
  from: string,
  params: Event
): Promise<{ world: World; comp: Comp; tokenData: TokenData }> {
  const fetchers = [
    new Fetcher<{ account: AddressV }, TokenData>(
      `
      #### Scenario

      * "Comp Deploy Scenario account:<Address>" - Deploys Scenario Comp Token
        * E.g. "Comp Deploy Scenario Geoff"
    `,
      'Scenario',
      [
        new Arg("account", getAddressV),
      ],
      async (world, { account }) => {
        return {
          invokation: await CompScenarioContract.deploy<CompScenario>(world, from, [account.val]),
          contract: 'CompScenario',
          symbol: 'COMP',
          name: 'Compound Governance Token',
          decimals: 18
        };
      }
    ),

    new Fetcher<{ account: AddressV }, TokenData>(
      `
      #### Comp

      * "Comp Deploy account:<Address>" - Deploys Comp Token
        * E.g. "Comp Deploy Geoff"
    `,
      'Comp',
      [
        new Arg("account", getAddressV),
      ],
      async (world, { account }) => {
        if (world.isLocalNetwork()) {
          return {
            invokation: await CompScenarioContract.deploy<CompScenario>(world, from, [account.val]),
            contract: 'CompScenario',
            symbol: 'COMP',
            name: 'Compound Governance Token',
            decimals: 18
          };
        } else {
          return {
            invokation: await CompContract.deploy<Comp>(world, from, [account.val]),
            contract: 'Comp',
            symbol: 'COMP',
            name: 'Compound Governance Token',
            decimals: 18
          };
        }
      },
      { catchall: true }
    )
  ];

  let tokenData = await getFetcherValue<any, TokenData>("DeployComp", fetchers, world, params);
  let invokation = tokenData.invokation;
  delete tokenData.invokation;

  if (invokation.error) {
    throw invokation.error;
  }

  const comp = invokation.value!;
  tokenData.address = comp._address;

  world = await storeAndSaveContract(
    world,
    comp,
    'Comp',
    invokation,
    [
      { index: ['Comp'], data: tokenData },
      { index: ['Tokens', tokenData.symbol], data: tokenData }
    ]
  );

  tokenData.invokation = invokation;

  return { world, comp, tokenData };
}
