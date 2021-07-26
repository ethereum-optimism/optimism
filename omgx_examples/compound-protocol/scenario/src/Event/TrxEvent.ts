import {World} from '../World';
import {Event} from '../Event';
import {processCoreEvent} from '../CoreEvent';
import {
  EventV,
  NumberV
} from '../Value';
import {
  getEventV,
  getNumberV
} from '../CoreValue';
import {Arg, Command, processCommandEvent} from '../Command';
import {encodedNumber} from '../Encoding';

async function setTrxValue(world: World, value: encodedNumber): Promise<World> {
	return world.update('trxInvokationOpts', (t) => t.set('value', value.toString()));
}

async function setTrxGasPrice(world: World, gasPrice: encodedNumber): Promise<World> {
  return world.update('trxInvokationOpts', (t) => t.set('gasPrice', gasPrice.toString()));;
}

export function trxCommands() {
  return [
    new Command<{amount: NumberV, event: EventV}>(`
        #### Value

        * "Value <Amount> <Event>" - Runs event with a set amount for any transactions
          * E.g. "Value 1.0e18 (CToken cEth Mint 1.0e18)"
      `,
      "Value",
      [
        new Arg("amount", getNumberV),
        new Arg("event", getEventV)
      ],
      async (world, from, {amount, event}) => processCoreEvent(await setTrxValue(world, amount.encode()), event.val, from)
    ),
    new Command<{gasPrice: NumberV, event: EventV}>(`
        #### GasPrice

        * "GasPrice <Amount> <Event>" - Runs event with a given gas price
          * E.g. "GasPrice 0 (CToken cEth Mint 1.0e18)"
      `,
      "GasPrice",
      [
        new Arg("gasPrice", getNumberV),
        new Arg("event", getEventV)
      ],
      async (world, from, {gasPrice, event}) => processCoreEvent(await setTrxGasPrice(world, gasPrice.encode()), event.val, from)
    )
  ];
}

export async function processTrxEvent(world: World, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>("Trx", trxCommands(), world, event, from);
}
