import { Event } from '../Event';
import { World } from '../World';
import { CErc20Delegate } from '../Contract/CErc20Delegate';
import {
  getCoreValue,
  mapValue
} from '../CoreValue';
import { Arg, Fetcher, getFetcherValue } from '../Command';
import {
  AddressV,
  Value,
} from '../Value';
import { getWorldContractByAddress, getCTokenDelegateAddress } from '../ContractLookup';

export async function getCTokenDelegateV(world: World, event: Event): Promise<CErc20Delegate> {
  const address = await mapValue<AddressV>(
    world,
    event,
    (str) => new AddressV(getCTokenDelegateAddress(world, str)),
    getCoreValue,
    AddressV
  );

  return getWorldContractByAddress<CErc20Delegate>(world, address.val);
}

async function cTokenDelegateAddress(world: World, cTokenDelegate: CErc20Delegate): Promise<AddressV> {
  return new AddressV(cTokenDelegate._address);
}

export function cTokenDelegateFetchers() {
  return [
    new Fetcher<{ cTokenDelegate: CErc20Delegate }, AddressV>(`
        #### Address

        * "CTokenDelegate <CTokenDelegate> Address" - Returns address of CTokenDelegate contract
          * E.g. "CTokenDelegate cDaiDelegate Address" - Returns cDaiDelegate's address
      `,
      "Address",
      [
        new Arg("cTokenDelegate", getCTokenDelegateV)
      ],
      (world, { cTokenDelegate }) => cTokenDelegateAddress(world, cTokenDelegate),
      { namePos: 1 }
    ),
  ];
}

export async function getCTokenDelegateValue(world: World, event: Event): Promise<Value> {
  return await getFetcherValue<any, any>("CTokenDelegate", cTokenDelegateFetchers(), world, event);
}
