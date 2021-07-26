import { Event } from '../Event';
import { World } from '../World';
import { getContract } from '../Contract';
import { Pot } from '../Contract/Pot';
import { Vat } from '../Contract/Vat';
import {
  getAddressV,
  getCoreValue,
  getStringV
} from '../CoreValue';
import { Arg, Fetcher, getFetcherValue } from '../Command';
import {
  AddressV,
  NumberV,
  Value,
  StringV
} from '../Value';

export function mcdFetchers() {
  return [
    new Fetcher<{ potAddress: AddressV, method: StringV, args: StringV[] }, Value>(`
        #### PotAt

        * "MCD PotAt <potAddress> <method> <args>"
          * E.g. "MCD PotAt "0xPotAddress" "pie" (CToken cDai Address)"
      `,
      "PotAt",
      [
        new Arg("potAddress", getAddressV),
        new Arg("method", getStringV),
        new Arg('args', getCoreValue, { variadic: true, mapped: true })
      ],
      async (world, { potAddress, method, args }) => {
        const PotContract = getContract('PotLike');
        const pot = await PotContract.at<Pot>(world, potAddress.val);
        const argStrings = args.map(arg => arg.val);
        return new NumberV(await pot.methods[method.val](...argStrings).call())
      }
    ),

    new Fetcher<{ vatAddress: AddressV, method: StringV, args: StringV[] }, Value>(`
        #### VatAt

        * "MCD VatAt <vatAddress> <method> <args>"
          * E.g. "MCD VatAt "0xVatAddress" "dai" (CToken cDai Address)"
      `,
      "VatAt",
      [
        new Arg("vatAddress", getAddressV),
        new Arg("method", getStringV),
        new Arg('args', getCoreValue, { variadic: true, mapped: true })
      ],
      async (world, { vatAddress, method, args }) => {
        const VatContract = getContract('VatLike');
        const vat = await VatContract.at<Vat>(world, vatAddress.val);
        const argStrings = args.map(arg => arg.val);
        return new NumberV(await vat.methods[method.val](...argStrings).call())
      }
    )
  ];
}

export async function getMCDValue(world: World, event: Event): Promise<Value> {
  return await getFetcherValue<any, any>("MCD", mcdFetchers(), world, event);
}
