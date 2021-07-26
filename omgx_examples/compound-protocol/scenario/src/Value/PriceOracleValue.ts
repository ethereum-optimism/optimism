import {Event} from '../Event';
import {World} from '../World';
import {PriceOracle} from '../Contract/PriceOracle';
import {
  getAddressV
} from '../CoreValue';
import {
  AddressV,
  NumberV,
  Value} from '../Value';
import {Arg, Fetcher, getFetcherValue} from '../Command';
import {getPriceOracle} from '../ContractLookup';

async function getPrice(world: World, priceOracle: PriceOracle, asset: string): Promise<NumberV> {
  return new NumberV(await priceOracle.methods.assetPrices(asset).call());
}

export async function getPriceOracleAddress(world: World, priceOracle: PriceOracle): Promise<AddressV> {
  return new AddressV(priceOracle._address);
}

export function priceOracleFetchers() {
  return [
    new Fetcher<{priceOracle: PriceOracle}, AddressV>(`
        #### Address

        * "Address" - Gets the address of the global price oracle
      `,
      "Address",
      [
        new Arg("priceOracle", getPriceOracle, {implicit: true})
      ],
      (world, {priceOracle}) => getPriceOracleAddress(world, priceOracle)
    ),
    new Fetcher<{priceOracle: PriceOracle, asset: AddressV}, NumberV>(`
        #### Price

        * "Price asset:<Address>" - Gets the price of the given asset
      `,
      "Price",
      [
        new Arg("priceOracle", getPriceOracle, {implicit: true}),
        new Arg("asset", getAddressV,)
      ],
      (world, {priceOracle, asset}) => getPrice(world, priceOracle, asset.val)
    )
  ];
}

export async function getPriceOracleValue(world: World, event: Event): Promise<Value> {
  return await getFetcherValue<any, any>("PriceOracle", priceOracleFetchers(), world, event);
}
