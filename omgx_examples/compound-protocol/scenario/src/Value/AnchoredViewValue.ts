import {Event} from '../Event';
import {World} from '../World';
import {AnchoredView} from '../Contract/AnchoredView';
import {getAddressV} from '../CoreValue';
import {AddressV, NumberV, Value} from '../Value';
import {Arg, Fetcher, getFetcherValue} from '../Command';
import {getAnchoredView} from '../ContractLookup';

export async function getAnchoredViewAddress(_: World, anchoredView: AnchoredView): Promise<AddressV> {
  return new AddressV(anchoredView._address);
}

async function getUnderlyingPrice(_: World, anchoredView: AnchoredView, asset: string): Promise<NumberV> {
  return new NumberV(await anchoredView.methods.getUnderlyingPrice(asset).call());
}

export function anchoredViewFetchers() {
  return [
    new Fetcher<{anchoredView: AnchoredView, asset: AddressV}, NumberV>(`
        #### UnderlyingPrice

        * "UnderlyingPrice asset:<Address>" - Gets the price of the given asset
      `,
      "UnderlyingPrice",
      [
        new Arg("anchoredView", getAnchoredView, {implicit: true}),
        new Arg("asset", getAddressV)
      ],
      (world, {anchoredView, asset}) => getUnderlyingPrice(world, anchoredView, asset.val)
    )
  ];
}

export async function getAnchoredViewValue(world: World, event: Event): Promise<Value> {
  return await getFetcherValue<any, any>("AnchoredView", anchoredViewFetchers(), world, event);
}
