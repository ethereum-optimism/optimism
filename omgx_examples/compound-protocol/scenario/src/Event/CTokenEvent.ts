import { Event } from '../Event';
import { addAction, describeUser, World } from '../World';
import { decodeCall, getPastEvents } from '../Contract';
import { CToken, CTokenScenario } from '../Contract/CToken';
import { CErc20Delegate } from '../Contract/CErc20Delegate'
import { CErc20Delegator } from '../Contract/CErc20Delegator'
import { invoke, Sendable } from '../Invokation';
import {
  getAddressV,
  getEventV,
  getExpNumberV,
  getNumberV,
  getStringV,
  getBoolV
} from '../CoreValue';
import {
  AddressV,
  BoolV,
  EventV,
  NothingV,
  NumberV,
  StringV
} from '../Value';
import { getContract } from '../Contract';
import { Arg, Command, View, processCommandEvent } from '../Command';
import { CTokenErrorReporter } from '../ErrorReporter';
import { getComptroller, getCTokenData } from '../ContractLookup';
import { getExpMantissa } from '../Encoding';
import { buildCToken } from '../Builder/CTokenBuilder';
import { verify } from '../Verify';
import { getLiquidity } from '../Value/ComptrollerValue';
import { encodedNumber } from '../Encoding';
import { getCTokenV, getCErc20DelegatorV } from '../Value/CTokenValue';

function showTrxValue(world: World): string {
  return new NumberV(world.trxInvokationOpts.get('value')).show();
}

async function genCToken(world: World, from: string, event: Event): Promise<World> {
  let { world: nextWorld, cToken, tokenData } = await buildCToken(world, from, event);
  world = nextWorld;

  world = addAction(
    world,
    `Added cToken ${tokenData.name} (${tokenData.contract}<decimals=${tokenData.decimals}>) at address ${cToken._address}`,
    tokenData.invokation
  );

  return world;
}

async function accrueInterest(world: World, from: string, cToken: CToken): Promise<World> {
  let invokation = await invoke(world, cToken.methods.accrueInterest(), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: Interest accrued`,
    invokation
  );

  return world;
}

async function mint(world: World, from: string, cToken: CToken, amount: NumberV | NothingV): Promise<World> {
  let invokation;
  let showAmount;

  if (amount instanceof NumberV) {
    showAmount = amount.show();
    invokation = await invoke(world, cToken.methods.mint(amount.encode()), from, CTokenErrorReporter);
  } else {
    showAmount = showTrxValue(world);
    invokation = await invoke(world, cToken.methods.mint(), from, CTokenErrorReporter);
  }

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} mints ${showAmount}`,
    invokation
  );

  return world;
}

async function redeem(world: World, from: string, cToken: CToken, tokens: NumberV): Promise<World> {
  let invokation = await invoke(world, cToken.methods.redeem(tokens.encode()), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} redeems ${tokens.show()} tokens`,
    invokation
  );

  return world;
}

async function redeemUnderlying(world: World, from: string, cToken: CToken, amount: NumberV): Promise<World> {
  let invokation = await invoke(world, cToken.methods.redeemUnderlying(amount.encode()), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} redeems ${amount.show()} underlying`,
    invokation
  );

  return world;
}

async function borrow(world: World, from: string, cToken: CToken, amount: NumberV): Promise<World> {
  let invokation = await invoke(world, cToken.methods.borrow(amount.encode()), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} borrows ${amount.show()}`,
    invokation
  );

  return world;
}

async function repayBorrow(world: World, from: string, cToken: CToken, amount: NumberV | NothingV): Promise<World> {
  let invokation;
  let showAmount;

  if (amount instanceof NumberV) {
    showAmount = amount.show();
    invokation = await invoke(world, cToken.methods.repayBorrow(amount.encode()), from, CTokenErrorReporter);
  } else {
    showAmount = showTrxValue(world);
    invokation = await invoke(world, cToken.methods.repayBorrow(), from, CTokenErrorReporter);
  }

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} repays ${showAmount} of borrow`,
    invokation
  );

  return world;
}

async function repayBorrowBehalf(world: World, from: string, behalf: string, cToken: CToken, amount: NumberV | NothingV): Promise<World> {
  let invokation;
  let showAmount;

  if (amount instanceof NumberV) {
    showAmount = amount.show();
    invokation = await invoke(world, cToken.methods.repayBorrowBehalf(behalf, amount.encode()), from, CTokenErrorReporter);
  } else {
    showAmount = showTrxValue(world);
    invokation = await invoke(world, cToken.methods.repayBorrowBehalf(behalf), from, CTokenErrorReporter);
  }

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} repays ${showAmount} of borrow on behalf of ${describeUser(world, behalf)}`,
    invokation
  );

  return world;
}

async function liquidateBorrow(world: World, from: string, cToken: CToken, borrower: string, collateral: CToken, repayAmount: NumberV | NothingV): Promise<World> {
  let invokation;
  let showAmount;

  if (repayAmount instanceof NumberV) {
    showAmount = repayAmount.show();
    invokation = await invoke(world, cToken.methods.liquidateBorrow(borrower, repayAmount.encode(), collateral._address), from, CTokenErrorReporter);
  } else {
    showAmount = showTrxValue(world);
    invokation = await invoke(world, cToken.methods.liquidateBorrow(borrower, collateral._address), from, CTokenErrorReporter);
  }

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} liquidates ${showAmount} from of ${describeUser(world, borrower)}, seizing ${collateral.name}.`,
    invokation
  );

  return world;
}

async function seize(world: World, from: string, cToken: CToken, liquidator: string, borrower: string, seizeTokens: NumberV): Promise<World> {
  let invokation = await invoke(world, cToken.methods.seize(liquidator, borrower, seizeTokens.encode()), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} initiates seizing ${seizeTokens.show()} to ${describeUser(world, liquidator)} from ${describeUser(world, borrower)}.`,
    invokation
  );

  return world;
}

async function evilSeize(world: World, from: string, cToken: CToken, treasure: CToken, liquidator: string, borrower: string, seizeTokens: NumberV): Promise<World> {
  let invokation = await invoke(world, cToken.methods.evilSeize(treasure._address, liquidator, borrower, seizeTokens.encode()), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} initiates illegal seizing ${seizeTokens.show()} to ${describeUser(world, liquidator)} from ${describeUser(world, borrower)}.`,
    invokation
  );

  return world;
}

async function setPendingAdmin(world: World, from: string, cToken: CToken, newPendingAdmin: string): Promise<World> {
  let invokation = await invoke(world, cToken.methods._setPendingAdmin(newPendingAdmin), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} sets pending admin to ${newPendingAdmin}`,
    invokation
  );

  return world;
}

async function acceptAdmin(world: World, from: string, cToken: CToken): Promise<World> {
  let invokation = await invoke(world, cToken.methods._acceptAdmin(), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} accepts admin`,
    invokation
  );

  return world;
}

async function addReserves(world: World, from: string, cToken: CToken, amount: NumberV): Promise<World> {
  let invokation = await invoke(world, cToken.methods._addReserves(amount.encode()), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} adds to reserves by ${amount.show()}`,
    invokation
  );

  return world;
}

async function reduceReserves(world: World, from: string, cToken: CToken, amount: NumberV): Promise<World> {
  let invokation = await invoke(world, cToken.methods._reduceReserves(amount.encode()), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} reduces reserves by ${amount.show()}`,
    invokation
  );

  return world;
}

async function setReserveFactor(world: World, from: string, cToken: CToken, reserveFactor: NumberV): Promise<World> {
  let invokation = await invoke(world, cToken.methods._setReserveFactor(reserveFactor.encode()), from, CTokenErrorReporter);

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(world, from)} sets reserve factor to ${reserveFactor.show()}`,
    invokation
  );

  return world;
}

async function setInterestRateModel(world: World, from: string, cToken: CToken, interestRateModel: string): Promise<World> {
  let invokation = await invoke(world, cToken.methods._setInterestRateModel(interestRateModel), from, CTokenErrorReporter);

  world = addAction(
    world,
    `Set interest rate for ${cToken.name} to ${interestRateModel} as ${describeUser(world, from)}`,
    invokation
  );

  return world;
}

async function setComptroller(world: World, from: string, cToken: CToken, comptroller: string): Promise<World> {
  let invokation = await invoke(world, cToken.methods._setComptroller(comptroller), from, CTokenErrorReporter);

  world = addAction(
    world,
    `Set comptroller for ${cToken.name} to ${comptroller} as ${describeUser(world, from)}`,
    invokation
  );

  return world;
}

async function sweepToken(world: World, from: string, cToken: CToken, token: string): Promise<World> {
  let invokation = await invoke(world, cToken.methods.sweepToken(token), from, CTokenErrorReporter);

  world = addAction(
    world,
    `Swept ERC-20 at ${token} to admin`,
    invokation
  );

  return world;
}

async function becomeImplementation(
  world: World,
  from: string,
  cToken: CToken,
  becomeImplementationData: string
): Promise<World> {

  const cErc20Delegate = getContract('CErc20Delegate');
  const cErc20DelegateContract = await cErc20Delegate.at<CErc20Delegate>(world, cToken._address);

  let invokation = await invoke(
    world,
    cErc20DelegateContract.methods._becomeImplementation(becomeImplementationData),
    from,
    CTokenErrorReporter
  );

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(
      world,
      from
    )} initiates _becomeImplementation with data:${becomeImplementationData}.`,
    invokation
  );

  return world;
}

async function resignImplementation(
  world: World,
  from: string,
  cToken: CToken,
): Promise<World> {

  const cErc20Delegate = getContract('CErc20Delegate');
  const cErc20DelegateContract = await cErc20Delegate.at<CErc20Delegate>(world, cToken._address);

  let invokation = await invoke(
    world,
    cErc20DelegateContract.methods._resignImplementation(),
    from,
    CTokenErrorReporter
  );

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(
      world,
      from
    )} initiates _resignImplementation.`,
    invokation
  );

  return world;
}

async function setImplementation(
  world: World,
  from: string,
  cToken: CErc20Delegator,
  implementation: string,
  allowResign: boolean,
  becomeImplementationData: string
): Promise<World> {
  let invokation = await invoke(
    world,
    cToken.methods._setImplementation(
      implementation,
      allowResign,
      becomeImplementationData
    ),
    from,
    CTokenErrorReporter
  );

  world = addAction(
    world,
    `CToken ${cToken.name}: ${describeUser(
      world,
      from
    )} initiates setImplementation with implementation:${implementation} allowResign:${allowResign} data:${becomeImplementationData}.`,
    invokation
  );

  return world;
}

async function donate(world: World, from: string, cToken: CToken): Promise<World> {
  let invokation = await invoke(world, cToken.methods.donate(), from, CTokenErrorReporter);

  world = addAction(
    world,
    `Donate for ${cToken.name} as ${describeUser(world, from)} with value ${showTrxValue(world)}`,
    invokation
  );

  return world;
}

async function setCTokenMock(world: World, from: string, cToken: CTokenScenario, mock: string, value: NumberV): Promise<World> {
  let mockMethod: (number) => Sendable<void>;

  switch (mock.toLowerCase()) {
    case "totalborrows":
      mockMethod = cToken.methods.setTotalBorrows;
      break;
    case "totalreserves":
      mockMethod = cToken.methods.setTotalReserves;
      break;
    default:
      throw new Error(`Mock "${mock}" not defined for cToken`);
  }

  let invokation = await invoke(world, mockMethod(value.encode()), from);

  world = addAction(
    world,
    `Mocked ${mock}=${value.show()} for ${cToken.name}`,
    invokation
  );

  return world;
}

async function verifyCToken(world: World, cToken: CToken, name: string, contract: string, apiKey: string): Promise<World> {
  if (world.isLocalNetwork()) {
    world.printer.printLine(`Politely declining to verify on local network: ${world.network}.`);
  } else {
    await verify(world, apiKey, name, contract, cToken._address);
  }

  return world;
}

async function printMinters(world: World, cToken: CToken): Promise<World> {
  let events = await getPastEvents(world, cToken, cToken.name, 'Mint');
  let addresses = events.map((event) => event.returnValues['minter']);
  let uniq = [...new Set(addresses)];

  world.printer.printLine("Minters:")

  uniq.forEach((address) => {
    world.printer.printLine(`\t${address}`)
  });

  return world;
}

async function printBorrowers(world: World, cToken: CToken): Promise<World> {
  let events = await getPastEvents(world, cToken, cToken.name, 'Borrow');
  let addresses = events.map((event) => event.returnValues['borrower']);
  let uniq = [...new Set(addresses)];

  world.printer.printLine("Borrowers:")

  uniq.forEach((address) => {
    world.printer.printLine(`\t${address}`)
  });

  return world;
}

async function printLiquidity(world: World, cToken: CToken): Promise<World> {
  let mintEvents = await getPastEvents(world, cToken, cToken.name, 'Mint');
  let mintAddresses = mintEvents.map((event) => event.returnValues['minter']);
  let borrowEvents = await getPastEvents(world, cToken, cToken.name, 'Borrow');
  let borrowAddresses = borrowEvents.map((event) => event.returnValues['borrower']);
  let uniq = [...new Set(mintAddresses.concat(borrowAddresses))];
  let comptroller = await getComptroller(world);

  world.printer.printLine("Liquidity:")

  const liquidityMap = await Promise.all(uniq.map(async (address) => {
    let userLiquidity = await getLiquidity(world, comptroller, address);

    return [address, userLiquidity.val];
  }));

  liquidityMap.forEach(([address, liquidity]) => {
    world.printer.printLine(`\t${world.settings.lookupAlias(address)}: ${liquidity / 1e18}e18`)
  });

  return world;
}

export function cTokenCommands() {
  return [
    new Command<{ cTokenParams: EventV }>(`
        #### Deploy

        * "CToken Deploy ...cTokenParams" - Generates a new CToken
          * E.g. "CToken cZRX Deploy"
      `,
      "Deploy",
      [new Arg("cTokenParams", getEventV, { variadic: true })],
      (world, from, { cTokenParams }) => genCToken(world, from, cTokenParams.val)
    ),
    new View<{ cTokenArg: StringV, apiKey: StringV }>(`
        #### Verify

        * "CToken <cToken> Verify apiKey:<String>" - Verifies CToken in Etherscan
          * E.g. "CToken cZRX Verify "myApiKey"
      `,
      "Verify",
      [
        new Arg("cTokenArg", getStringV),
        new Arg("apiKey", getStringV)
      ],
      async (world, { cTokenArg, apiKey }) => {
        let [cToken, name, data] = await getCTokenData(world, cTokenArg.val);

        return await verifyCToken(world, cToken, name, data.get('contract')!, apiKey.val);
      },
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken }>(`
        #### AccrueInterest

        * "CToken <cToken> AccrueInterest" - Accrues interest for given token
          * E.g. "CToken cZRX AccrueInterest"
      `,
      "AccrueInterest",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, from, { cToken }) => accrueInterest(world, from, cToken),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, amount: NumberV | NothingV }>(`
        #### Mint

        * "CToken <cToken> Mint amount:<Number>" - Mints the given amount of cToken as specified user
          * E.g. "CToken cZRX Mint 1.0e18"
      `,
      "Mint",
      [
        new Arg("cToken", getCTokenV),
        new Arg("amount", getNumberV, { nullable: true })
      ],
      (world, from, { cToken, amount }) => mint(world, from, cToken, amount),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, tokens: NumberV }>(`
        #### Redeem

        * "CToken <cToken> Redeem tokens:<Number>" - Redeems the given amount of cTokens as specified user
          * E.g. "CToken cZRX Redeem 1.0e9"
      `,
      "Redeem",
      [
        new Arg("cToken", getCTokenV),
        new Arg("tokens", getNumberV)
      ],
      (world, from, { cToken, tokens }) => redeem(world, from, cToken, tokens),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, amount: NumberV }>(`
        #### RedeemUnderlying

        * "CToken <cToken> RedeemUnderlying amount:<Number>" - Redeems the given amount of underlying as specified user
          * E.g. "CToken cZRX RedeemUnderlying 1.0e18"
      `,
      "RedeemUnderlying",
      [
        new Arg("cToken", getCTokenV),
        new Arg("amount", getNumberV)
      ],
      (world, from, { cToken, amount }) => redeemUnderlying(world, from, cToken, amount),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, amount: NumberV }>(`
        #### Borrow

        * "CToken <cToken> Borrow amount:<Number>" - Borrows the given amount of this cToken as specified user
          * E.g. "CToken cZRX Borrow 1.0e18"
      `,
      "Borrow",
      [
        new Arg("cToken", getCTokenV),
        new Arg("amount", getNumberV)
      ],
      // Note: we override from
      (world, from, { cToken, amount }) => borrow(world, from, cToken, amount),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, amount: NumberV | NothingV }>(`
        #### RepayBorrow

        * "CToken <cToken> RepayBorrow underlyingAmount:<Number>" - Repays borrow in the given underlying amount as specified user
          * E.g. "CToken cZRX RepayBorrow 1.0e18"
      `,
      "RepayBorrow",
      [
        new Arg("cToken", getCTokenV),
        new Arg("amount", getNumberV, { nullable: true })
      ],
      (world, from, { cToken, amount }) => repayBorrow(world, from, cToken, amount),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, behalf: AddressV, amount: NumberV | NothingV }>(`
        #### RepayBorrowBehalf

        * "CToken <cToken> RepayBorrowBehalf behalf:<User> underlyingAmount:<Number>" - Repays borrow in the given underlying amount on behalf of another user
          * E.g. "CToken cZRX RepayBorrowBehalf Geoff 1.0e18"
      `,
      "RepayBorrowBehalf",
      [
        new Arg("cToken", getCTokenV),
        new Arg("behalf", getAddressV),
        new Arg("amount", getNumberV, { nullable: true })
      ],
      (world, from, { cToken, behalf, amount }) => repayBorrowBehalf(world, from, behalf.val, cToken, amount),
      { namePos: 1 }
    ),
    new Command<{ borrower: AddressV, cToken: CToken, collateral: CToken, repayAmount: NumberV | NothingV }>(`
        #### Liquidate

        * "CToken <cToken> Liquidate borrower:<User> cTokenCollateral:<Address> repayAmount:<Number>" - Liquidates repayAmount of given token seizing collateral token
          * E.g. "CToken cZRX Liquidate Geoff cBAT 1.0e18"
      `,
      "Liquidate",
      [
        new Arg("cToken", getCTokenV),
        new Arg("borrower", getAddressV),
        new Arg("collateral", getCTokenV),
        new Arg("repayAmount", getNumberV, { nullable: true })
      ],
      (world, from, { borrower, cToken, collateral, repayAmount }) => liquidateBorrow(world, from, cToken, borrower.val, collateral, repayAmount),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, liquidator: AddressV, borrower: AddressV, seizeTokens: NumberV }>(`
        #### Seize

        * "CToken <cToken> Seize liquidator:<User> borrower:<User> seizeTokens:<Number>" - Seizes a given number of tokens from a user (to be called from other CToken)
          * E.g. "CToken cZRX Seize Geoff Torrey 1.0e18"
      `,
      "Seize",
      [
        new Arg("cToken", getCTokenV),
        new Arg("liquidator", getAddressV),
        new Arg("borrower", getAddressV),
        new Arg("seizeTokens", getNumberV)
      ],
      (world, from, { cToken, liquidator, borrower, seizeTokens }) => seize(world, from, cToken, liquidator.val, borrower.val, seizeTokens),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, treasure: CToken, liquidator: AddressV, borrower: AddressV, seizeTokens: NumberV }>(`
        #### EvilSeize

        * "CToken <cToken> EvilSeize treasure:<Token> liquidator:<User> borrower:<User> seizeTokens:<Number>" - Improperly seizes a given number of tokens from a user
          * E.g. "CToken cEVL EvilSeize cZRX Geoff Torrey 1.0e18"
      `,
      "EvilSeize",
      [
        new Arg("cToken", getCTokenV),
        new Arg("treasure", getCTokenV),
        new Arg("liquidator", getAddressV),
        new Arg("borrower", getAddressV),
        new Arg("seizeTokens", getNumberV)
      ],
      (world, from, { cToken, treasure, liquidator, borrower, seizeTokens }) => evilSeize(world, from, cToken, treasure, liquidator.val, borrower.val, seizeTokens),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, amount: NumberV }>(`
        #### ReduceReserves

        * "CToken <cToken> ReduceReserves amount:<Number>" - Reduces the reserves of the cToken
          * E.g. "CToken cZRX ReduceReserves 1.0e18"
      `,
      "ReduceReserves",
      [
        new Arg("cToken", getCTokenV),
        new Arg("amount", getNumberV)
      ],
      (world, from, { cToken, amount }) => reduceReserves(world, from, cToken, amount),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, amount: NumberV }>(`
    #### AddReserves

    * "CToken <cToken> AddReserves amount:<Number>" - Adds reserves to the cToken
      * E.g. "CToken cZRX AddReserves 1.0e18"
  `,
      "AddReserves",
      [
        new Arg("cToken", getCTokenV),
        new Arg("amount", getNumberV)
      ],
      (world, from, { cToken, amount }) => addReserves(world, from, cToken, amount),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, newPendingAdmin: AddressV }>(`
        #### SetPendingAdmin

        * "CToken <cToken> SetPendingAdmin newPendingAdmin:<Address>" - Sets the pending admin for the cToken
          * E.g. "CToken cZRX SetPendingAdmin Geoff"
      `,
      "SetPendingAdmin",
      [
        new Arg("cToken", getCTokenV),
        new Arg("newPendingAdmin", getAddressV)
      ],
      (world, from, { cToken, newPendingAdmin }) => setPendingAdmin(world, from, cToken, newPendingAdmin.val),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken }>(`
        #### AcceptAdmin

        * "CToken <cToken> AcceptAdmin" - Accepts admin for the cToken
          * E.g. "From Geoff (CToken cZRX AcceptAdmin)"
      `,
      "AcceptAdmin",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, from, { cToken }) => acceptAdmin(world, from, cToken),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, reserveFactor: NumberV }>(`
        #### SetReserveFactor

        * "CToken <cToken> SetReserveFactor reserveFactor:<Number>" - Sets the reserve factor for the cToken
          * E.g. "CToken cZRX SetReserveFactor 0.1"
      `,
      "SetReserveFactor",
      [
        new Arg("cToken", getCTokenV),
        new Arg("reserveFactor", getExpNumberV)
      ],
      (world, from, { cToken, reserveFactor }) => setReserveFactor(world, from, cToken, reserveFactor),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, interestRateModel: AddressV }>(`
        #### SetInterestRateModel

        * "CToken <cToken> SetInterestRateModel interestRateModel:<Contract>" - Sets the interest rate model for the given cToken
          * E.g. "CToken cZRX SetInterestRateModel (FixedRate 1.5)"
      `,
      "SetInterestRateModel",
      [
        new Arg("cToken", getCTokenV),
        new Arg("interestRateModel", getAddressV)
      ],
      (world, from, { cToken, interestRateModel }) => setInterestRateModel(world, from, cToken, interestRateModel.val),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, token: AddressV }>(`
        #### SweepToken

        * "CToken <cToken> SweepToken erc20Token:<Contract>" - Sweeps the given erc-20 token from the contract
          * E.g. "CToken cZRX SweepToken BAT"
      `,
      "SweepToken",
      [
        new Arg("cToken", getCTokenV),
        new Arg("token", getAddressV)
      ],
      (world, from, { cToken, token }) => sweepToken(world, from, cToken, token.val),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, comptroller: AddressV }>(`
        #### SetComptroller

        * "CToken <cToken> SetComptroller comptroller:<Contract>" - Sets the comptroller for the given cToken
          * E.g. "CToken cZRX SetComptroller Comptroller"
      `,
      "SetComptroller",
      [
        new Arg("cToken", getCTokenV),
        new Arg("comptroller", getAddressV)
      ],
      (world, from, { cToken, comptroller }) => setComptroller(world, from, cToken, comptroller.val),
      { namePos: 1 }
    ),
    new Command<{
      cToken: CToken;
      becomeImplementationData: StringV;
    }>(
      `
        #### BecomeImplementation

        * "CToken <cToken> BecomeImplementation becomeImplementationData:<String>"
          * E.g. "CToken cDAI BecomeImplementation "0x01234anyByTeS56789""
      `,
      'BecomeImplementation',
      [
        new Arg('cToken', getCTokenV),
        new Arg('becomeImplementationData', getStringV)
      ],
      (world, from, { cToken, becomeImplementationData }) =>
        becomeImplementation(
          world,
          from,
          cToken,
          becomeImplementationData.val
        ),
      { namePos: 1 }
    ),
    new Command<{cToken: CToken;}>(
      `
        #### ResignImplementation

        * "CToken <cToken> ResignImplementation"
          * E.g. "CToken cDAI ResignImplementation"
      `,
      'ResignImplementation',
      [new Arg('cToken', getCTokenV)],
      (world, from, { cToken }) =>
        resignImplementation(
          world,
          from,
          cToken
        ),
      { namePos: 1 }
    ),
    new Command<{
      cToken: CErc20Delegator;
      implementation: AddressV;
      allowResign: BoolV;
      becomeImplementationData: StringV;
    }>(
      `
        #### SetImplementation

        * "CToken <cToken> SetImplementation implementation:<Address> allowResign:<Bool> becomeImplementationData:<String>"
          * E.g. "CToken cDAI SetImplementation (CToken cDAIDelegate Address) True "0x01234anyByTeS56789"
      `,
      'SetImplementation',
      [
        new Arg('cToken', getCErc20DelegatorV),
        new Arg('implementation', getAddressV),
        new Arg('allowResign', getBoolV),
        new Arg('becomeImplementationData', getStringV)
      ],
      (world, from, { cToken, implementation, allowResign, becomeImplementationData }) =>
        setImplementation(
          world,
          from,
          cToken,
          implementation.val,
          allowResign.val,
          becomeImplementationData.val
        ),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken }>(`
        #### Donate

        * "CToken <cToken> Donate" - Calls the donate (payable no-op) function
          * E.g. "(Trx Value 5.0e18 (CToken cETH Donate))"
      `,
      "Donate",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, from, { cToken }) => donate(world, from, cToken),
      { namePos: 1 }
    ),
    new Command<{ cToken: CToken, variable: StringV, value: NumberV }>(`
        #### Mock

        * "CToken <cToken> Mock variable:<String> value:<Number>" - Mocks a given value on cToken. Note: value must be a supported mock and this will only work on a "CTokenScenario" contract.
          * E.g. "CToken cZRX Mock totalBorrows 5.0e18"
          * E.g. "CToken cZRX Mock totalReserves 0.5e18"
      `,
      "Mock",
      [
        new Arg("cToken", getCTokenV),
        new Arg("variable", getStringV),
        new Arg("value", getNumberV),
      ],
      (world, from, { cToken, variable, value }) => setCTokenMock(world, from, <CTokenScenario>cToken, variable.val, value),
      { namePos: 1 }
    ),
    new View<{ cToken: CToken }>(`
        #### Minters

        * "CToken <cToken> Minters" - Print address of all minters
      `,
      "Minters",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => printMinters(world, cToken),
      { namePos: 1 }
    ),
    new View<{ cToken: CToken }>(`
        #### Borrowers

        * "CToken <cToken> Borrowers" - Print address of all borrowers
      `,
      "Borrowers",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => printBorrowers(world, cToken),
      { namePos: 1 }
    ),
    new View<{ cToken: CToken }>(`
        #### Liquidity

        * "CToken <cToken> Liquidity" - Prints liquidity of all minters or borrowers
      `,
      "Liquidity",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => printLiquidity(world, cToken),
      { namePos: 1 }
    ),
    new View<{ cToken: CToken, input: StringV }>(`
        #### Decode

        * "Decode <cToken> input:<String>" - Prints information about a call to a cToken contract
      `,
      "Decode",
      [
        new Arg("cToken", getCTokenV),
        new Arg("input", getStringV)

      ],
      (world, { cToken, input }) => decodeCall(world, cToken, input.val),
      { namePos: 1 }
    )
  ];
}

export async function processCTokenEvent(world: World, event: Event, from: string | null): Promise<World> {
  return await processCommandEvent<any>("CToken", cTokenCommands(), world, event, from);
}
