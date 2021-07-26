import { Event } from '../Event';
import { World } from '../World';
import { CToken } from '../Contract/CToken';
import { CErc20Delegator } from '../Contract/CErc20Delegator';
import { Erc20 } from '../Contract/Erc20';
import {
  getAddressV,
  getCoreValue,
  getStringV,
  mapValue
} from '../CoreValue';
import { Arg, Fetcher, getFetcherValue } from '../Command';
import {
  AddressV,
  NumberV,
  Value,
  StringV
} from '../Value';
import { getWorldContractByAddress, getCTokenAddress } from '../ContractLookup';

export async function getCTokenV(world: World, event: Event): Promise<CToken> {
  const address = await mapValue<AddressV>(
    world,
    event,
    (str) => new AddressV(getCTokenAddress(world, str)),
    getCoreValue,
    AddressV
  );

  return getWorldContractByAddress<CToken>(world, address.val);
}

export async function getCErc20DelegatorV(world: World, event: Event): Promise<CErc20Delegator> {
  const address = await mapValue<AddressV>(
    world,
    event,
    (str) => new AddressV(getCTokenAddress(world, str)),
    getCoreValue,
    AddressV
  );

  return getWorldContractByAddress<CErc20Delegator>(world, address.val);
}

async function getInterestRateModel(world: World, cToken: CToken): Promise<AddressV> {
  return new AddressV(await cToken.methods.interestRateModel().call());
}

async function cTokenAddress(world: World, cToken: CToken): Promise<AddressV> {
  return new AddressV(cToken._address);
}

async function getCTokenAdmin(world: World, cToken: CToken): Promise<AddressV> {
  return new AddressV(await cToken.methods.admin().call());
}

async function getCTokenPendingAdmin(world: World, cToken: CToken): Promise<AddressV> {
  return new AddressV(await cToken.methods.pendingAdmin().call());
}

async function balanceOfUnderlying(world: World, cToken: CToken, user: string): Promise<NumberV> {
  return new NumberV(await cToken.methods.balanceOfUnderlying(user).call());
}

async function getBorrowBalance(world: World, cToken: CToken, user): Promise<NumberV> {
  return new NumberV(await cToken.methods.borrowBalanceCurrent(user).call());
}

async function getBorrowBalanceStored(world: World, cToken: CToken, user): Promise<NumberV> {
  return new NumberV(await cToken.methods.borrowBalanceStored(user).call());
}

async function getTotalBorrows(world: World, cToken: CToken): Promise<NumberV> {
  return new NumberV(await cToken.methods.totalBorrows().call());
}

async function getTotalBorrowsCurrent(world: World, cToken: CToken): Promise<NumberV> {
  return new NumberV(await cToken.methods.totalBorrowsCurrent().call());
}

async function getReserveFactor(world: World, cToken: CToken): Promise<NumberV> {
  return new NumberV(await cToken.methods.reserveFactorMantissa().call(), 1.0e18);
}

async function getTotalReserves(world: World, cToken: CToken): Promise<NumberV> {
  return new NumberV(await cToken.methods.totalReserves().call());
}

async function getComptroller(world: World, cToken: CToken): Promise<AddressV> {
  return new AddressV(await cToken.methods.comptroller().call());
}

async function getExchangeRateStored(world: World, cToken: CToken): Promise<NumberV> {
  return new NumberV(await cToken.methods.exchangeRateStored().call());
}

async function getExchangeRate(world: World, cToken: CToken): Promise<NumberV> {
  return new NumberV(await cToken.methods.exchangeRateCurrent().call(), 1e18);
}

async function getCash(world: World, cToken: CToken): Promise<NumberV> {
  return new NumberV(await cToken.methods.getCash().call());
}

async function getInterestRate(world: World, cToken: CToken): Promise<NumberV> {
  return new NumberV(await cToken.methods.borrowRatePerBlock().call(), 1.0e18 / 2102400);
}

async function getImplementation(world: World, cToken: CToken): Promise<AddressV> {
  return new AddressV(await (cToken as CErc20Delegator).methods.implementation().call());
}

export function cTokenFetchers() {
  return [
    new Fetcher<{ cToken: CToken }, AddressV>(`
        #### Address

        * "CToken <CToken> Address" - Returns address of CToken contract
          * E.g. "CToken cZRX Address" - Returns cZRX's address
      `,
      "Address",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => cTokenAddress(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, AddressV>(`
        #### InterestRateModel

        * "CToken <CToken> InterestRateModel" - Returns the interest rate model of CToken contract
          * E.g. "CToken cZRX InterestRateModel" - Returns cZRX's interest rate model
      `,
      "InterestRateModel",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getInterestRateModel(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, AddressV>(`
        #### Admin

        * "CToken <CToken> Admin" - Returns the admin of CToken contract
          * E.g. "CToken cZRX Admin" - Returns cZRX's admin
      `,
      "Admin",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getCTokenAdmin(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, AddressV>(`
        #### PendingAdmin

        * "CToken <CToken> PendingAdmin" - Returns the pending admin of CToken contract
          * E.g. "CToken cZRX PendingAdmin" - Returns cZRX's pending admin
      `,
      "PendingAdmin",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getCTokenPendingAdmin(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, AddressV>(`
        #### Underlying

        * "CToken <CToken> Underlying" - Returns the underlying asset (if applicable)
          * E.g. "CToken cZRX Underlying"
      `,
      "Underlying",
      [
        new Arg("cToken", getCTokenV)
      ],
      async (world, { cToken }) => new AddressV(await cToken.methods.underlying().call()),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken, address: AddressV }, NumberV>(`
        #### UnderlyingBalance

        * "CToken <CToken> UnderlyingBalance <User>" - Returns a user's underlying balance (based on given exchange rate)
          * E.g. "CToken cZRX UnderlyingBalance Geoff"
      `,
      "UnderlyingBalance",
      [
        new Arg("cToken", getCTokenV),
        new Arg<AddressV>("address", getAddressV)
      ],
      (world, { cToken, address }) => balanceOfUnderlying(world, cToken, address.val),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken, address: AddressV }, NumberV>(`
        #### BorrowBalance

        * "CToken <CToken> BorrowBalance <User>" - Returns a user's borrow balance (including interest)
          * E.g. "CToken cZRX BorrowBalance Geoff"
      `,
      "BorrowBalance",
      [
        new Arg("cToken", getCTokenV),
        new Arg("address", getAddressV)
      ],
      (world, { cToken, address }) => getBorrowBalance(world, cToken, address.val),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken, address: AddressV }, NumberV>(`
        #### BorrowBalanceStored

        * "CToken <CToken> BorrowBalanceStored <User>" - Returns a user's borrow balance (without specifically re-accruing interest)
          * E.g. "CToken cZRX BorrowBalanceStored Geoff"
      `,
      "BorrowBalanceStored",
      [
        new Arg("cToken", getCTokenV),
        new Arg("address", getAddressV)
      ],
      (world, { cToken, address }) => getBorrowBalanceStored(world, cToken, address.val),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, NumberV>(`
        #### TotalBorrows

        * "CToken <CToken> TotalBorrows" - Returns the cToken's total borrow balance
          * E.g. "CToken cZRX TotalBorrows"
      `,
      "TotalBorrows",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getTotalBorrows(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, NumberV>(`
        #### TotalBorrowsCurrent

        * "CToken <CToken> TotalBorrowsCurrent" - Returns the cToken's total borrow balance with interest
          * E.g. "CToken cZRX TotalBorrowsCurrent"
      `,
      "TotalBorrowsCurrent",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getTotalBorrowsCurrent(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, NumberV>(`
        #### Reserves

        * "CToken <CToken> Reserves" - Returns the cToken's total reserves
          * E.g. "CToken cZRX Reserves"
      `,
      "Reserves",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getTotalReserves(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, NumberV>(`
        #### ReserveFactor

        * "CToken <CToken> ReserveFactor" - Returns reserve factor of CToken contract
          * E.g. "CToken cZRX ReserveFactor" - Returns cZRX's reserve factor
      `,
      "ReserveFactor",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getReserveFactor(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, AddressV>(`
        #### Comptroller

        * "CToken <CToken> Comptroller" - Returns the cToken's comptroller
          * E.g. "CToken cZRX Comptroller"
      `,
      "Comptroller",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getComptroller(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, NumberV>(`
        #### ExchangeRateStored

        * "CToken <CToken> ExchangeRateStored" - Returns the cToken's exchange rate (based on balances stored)
          * E.g. "CToken cZRX ExchangeRateStored"
      `,
      "ExchangeRateStored",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getExchangeRateStored(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, NumberV>(`
        #### ExchangeRate

        * "CToken <CToken> ExchangeRate" - Returns the cToken's current exchange rate
          * E.g. "CToken cZRX ExchangeRate"
      `,
      "ExchangeRate",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getExchangeRate(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, NumberV>(`
        #### Cash

        * "CToken <CToken> Cash" - Returns the cToken's current cash
          * E.g. "CToken cZRX Cash"
      `,
      "Cash",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getCash(world, cToken),
      { namePos: 1 }
    ),

    new Fetcher<{ cToken: CToken }, NumberV>(`
        #### InterestRate

        * "CToken <CToken> InterestRate" - Returns the cToken's current interest rate
          * E.g. "CToken cZRX InterestRate"
      `,
      "InterestRate",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, {cToken}) => getInterestRate(world, cToken),
      {namePos: 1}
    ),
    new Fetcher<{cToken: CToken, signature: StringV}, NumberV>(`
        #### CallNum

        * "CToken <CToken> Call <signature>" - Simple direct call method, for now with no parameters
          * E.g. "CToken cZRX Call \"borrowIndex()\""
      `,
      "CallNum",
      [
        new Arg("cToken", getCTokenV),
        new Arg("signature", getStringV),
      ],
      async (world, {cToken, signature}) => {
        const res = await world.web3.eth.call({
            to: cToken._address,
            data: world.web3.eth.abi.encodeFunctionSignature(signature.val)
          })
        const resNum : any = world.web3.eth.abi.decodeParameter('uint256',res);
        return new NumberV(resNum);
      }
      ,
      {namePos: 1}
    ),
    new Fetcher<{ cToken: CToken }, AddressV>(`
        #### Implementation

        * "CToken <CToken> Implementation" - Returns the cToken's current implementation
          * E.g. "CToken cDAI Implementation"
      `,
      "Implementation",
      [
        new Arg("cToken", getCTokenV)
      ],
      (world, { cToken }) => getImplementation(world, cToken),
      { namePos: 1 }
    )
  ];
}

export async function getCTokenValue(world: World, event: Event): Promise<Value> {
  return await getFetcherValue<any, any>("cToken", cTokenFetchers(), world, event);
}
