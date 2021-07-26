
# Types
* `name:<Type>` - Helper to describe arguments with names, not actually input this way
* `<Bool>` - `True` or `False`
* `<Number>` - A standard number (e.g. `5` or `6.0` or `10.0e18`)
* `<CToken>` - The local name for a given cToken when created, e.g. `cZRX`
* `<User>` - One of: `Admin, Bank, Geoff, Torrey, Robert, Coburn, Jared`
* `<String>` - A string, may be quoted but does not have to be if a single-word (e.g. `"Mint"` or `Mint`)
* `<Address>` - TODO
* `<Assertion>` - See assertions below.

# Events

## Core Events

* "History n:<Number>=5" - Prints history of actions
  * E.g. "History"
  * E.g. "History 10"
* `Read ...` - Reads given value and prints result
  * E.g. `Read CToken cBAT ExchangeRateStored` - Returns exchange rate of cBAT
* `Assert <Assertion>` - Validates given assertion, raising an exception if assertion fails
  * E.g. `Assert Equal (Erc20 BAT TokenBalance Geoff) (Exactly 5.0)` - Returns exchange rate of cBAT
* `FastForward n:<Number> Blocks` - For `CTokenScenario`, moves the block number forward n blocks. Note: in `CTokenScenario` the current block number is mocked (starting at 100000). Thus, this is the only way for the protocol to see a higher block number (for accruing interest).
  * E.g. `FastForward 5 Blocks` - Move block number forward 5 blocks.
* `Inspect` - Prints debugging information about the world
* `Debug message:<String>` - Same as inspect but prepends with a string
* `From <User> <Event>` - Runs event as the given user
  * E.g. `From Geoff (CToken cZRX Mint 5e18)`
* `Invariant <Invariant>` - Adds a new invariant to the world which is checked after each transaction
  * E.g. `Invariant Static (CToken cZRX TotalSupply)`
* `WipeInvariants` - Removes all invariants.
* `Comptroller <ComptrollerEvent>` - Runs given Comptroller event
  * E.g. `Comptroller _setReserveFactor 0.5`
* `CToken <CTokenEvent>` - Runs given CToken event
  * E.g. `CToken cZRX Mint 5e18`
* `Erc20 <Erc20Event>` - Runs given Erc20 event
  * E.g. `Erc20 ZRX Facuet Geoff 5e18`
* `InterestRateModel ...event` - Runs given interest rate model event
  * E.g. `InterestRateModel Deployed (Fixed 0.5)`
* `PriceOracle <PriceOracleEvent>` - Runs given Price Oracle event
  * E.g. `PriceOracle SetPrice cZRX 1.5`

## Comptroller Events

* "Comptroller Deploy ...comptrollerParams" - Generates a new Comptroller
  * E.g. "Comptroller Deploy Scenario (PriceOracle Address) 0.1 10"
* `Comptroller SetPaused action:<String> paused:<Bool>` - Pauses or unpaused given cToken function (e.g. Mint)
  * E.g. `Comptroller SetPaused Mint True`
* `Comptroller SupportMarket <CToken>` - Adds support in the Comptroller for the given cToken
  * E.g. `Comptroller SupportMarket cZRX`
* `Comptroller EnterMarkets <User> <CToken> ...` - User enters the given markets
  * E.g. `Comptroller EnterMarkets Geoff cZRX cETH`
* `Comptroller SetMaxAssets <Number>` - Sets (or resets) the max allowed asset count
  * E.g. `Comptroller SetMaxAssets 4`
* `CToken <cToken> SetOracle oracle:<Contract>` - Sets the oracle
  * E.g. `Comptroller SetOracle (Fixed 1.5)`
* `Comptroller SetCollateralFactor <CToken> <Number>` - Sets the collateral factor for given cToken to number
  * E.g. `Comptroller SetCollateralFactor cZRX 0.1`
* `FastForward n:<Number> Blocks` - Moves the block number forward `n` blocks. Note: in `CTokenScenario` and `ComptrollerScenario` the current block number is mocked (starting at 100000). This is the only way for the protocol to see a higher block number (for accruing interest).
  * E.g. `Comptroller FastForward 5 Blocks` - Move block number forward 5 blocks.

## cToken Events

* `CToken Deploy name:<CToken> underlying:<Contract> comptroller:<Contract> interestRateModel:<Contract> initialExchangeRate:<Number> decimals:<Number> admin:<Address>` - Generates a new comptroller and sets to world global
  * E.g. `CToken Deploy cZRX (Erc20 ZRX Address) (Comptroller Address) (InterestRateModel Address) 1.0 18`
* `CToken <cToken> AccrueInterest` - Accrues interest for given token
  * E.g. `CToken cZRX AccrueInterest`
* `CToken <cToken> Mint <User> amount:<Number>` - Mints the given amount of cToken as specified user
  * E.g. `CToken cZRX Mint Geoff 1.0`
* `CToken <cToken> Redeem <User> amount:<Number>` - Redeems the given amount of cToken as specified user
      * E.g. `CToken cZRX Redeem Geoff 1.0e18`
* `CToken <cToken> Borrow <User> amount:<Number>` - Borrows the given amount of this cToken as specified user
      * E.g. `CToken cZRX Borrow Geoff 1.0e18`
* `CToken <cToken> ReduceReserves amount:<Number>` - Reduces the reserves of the cToken
      * E.g. `CToken cZRX ReduceReserves 1.0e18`
* `CToken <cToken> SetReserveFactor amount:<Number>` - Sets the reserve factor for the cToken
      * E.g. `CToken cZRX SetReserveFactor 0.1`
* `CToken <cToken> SetInterestRateModel interestRateModel:<Contract>` - Sets the interest rate model for the given cToken
  * E.g. `CToken cZRX SetInterestRateModel (Fixed 1.5)`
* `CToken <cToken> SetComptroller comptroller:<Contract>` - Sets the comptroller for the given cToken
  * E.g. `CToken cZRX SetComptroller Comptroller`
* `CToken <cToken> Mock variable:<String> value:<Number>` - Mocks a given value on cToken. Note: value must be a supported mock and this will only work on a CTokenScenario contract.
  * E.g. `CToken cZRX Mock totalBorrows 5.0e18`
  * E.g. `CToken cZRX Mock totalReserves 0.5e18`

## Erc-20 Events

* `Erc20 Deploy name:<Erc20>` - Generates a new ERC-20 token by name
  * E.g. `Erc20 Deploy ZRX`
* `Erc20 <Erc20> Approve <User> <Address> <Amount>` - Adds an allowance between user and address
  * E.g. `Erc20 ZRX Approve Geoff cZRX 1.0e18`
* `Erc20 <Erc20> Faucet <Address> <Amount>` - Adds an arbitrary balance to given user
  * E.g. `Erc20 ZRX Facuet Geoff 1.0e18`

## Price Oracle Events

* `Deploy` - Generates a new price oracle (note: defaults to (Fixed 1.0))
  * E.g. `PriceOracle Deploy (Fixed 1.0)`
  * E.g. `PriceOracle Deploy Simple`
  * E.g. `PriceOracle Deploy NotPriceOracle`
* `SetPrice <CToken> <Amount>` - Sets the per-ether price for the given cToken
  * E.g. `PriceOracle SetPrice cZRX 1.0`

## Interest Rate Model Events

## Deploy

* `Deploy params:<String[]>` - Generates a new interest rate model (note: defaults to (Fixed 0.25))
  * E.g. `InterestRateModel Deploy (Fixed 0.5)`
  * E.g. `InterestRateModel Deploy Whitepaper`

# Values

## Core Values

* `True` - Returns true
* `False` - Returns false
* `Zero` - Returns 0
* `Some` - Returns 100e18
* `Little` - Returns 100e10
* `Exactly <Amount>` - Returns a strict numerical value
  * E.g. `Exactly 5.0`
* `Exp <Amount>` - Returns the mantissa for a given exp
  * E.g. `Exp 5.5`
* `Precisely <Amount>` - Matches a number to given number of significant figures
  * E.g. `Exactly 5.1000` - Matches to 5 sig figs
* `Anything` - Matches anything
* `Nothing` - Matches nothing
* `Default value:<Value> default:<Value>` - Returns value if truthy, otherwise default. Note: this does short-circuit
* `LastContract` - Returns the address of last constructed contract
* `User <...>` - Returns User value (see below)
* `Comptroller <...>` - Returns Comptroller value (see below)
* `CToken <...>` - Returns CToken value (see below)
* `Erc20 <...>` - Returns Erc20 value (see below)
* `InterestRateModel <...>` - Returns InterestRateModel value (see below)
* `PriceOracle <...>` - Returns PriceOracle value (see below)

## User Values

* `User <User> Address` - Returns address of user
  * E.g. `User Geoff Address` - Returns Geoff's address

## Comptroller Values

* `Comptroller Liquidity <User>` - Returns a given user's trued up liquidity
  * E.g. `Comptroller Liquidity Geoff`
* `Comptroller MembershipLength <User>` - Returns a given user's length of membership
  * E.g. `Comptroller MembershipLength Geoff`
* `Comptroller CheckMembership <User> <CToken>` - Returns one if user is in asset, zero otherwise.
  * E.g. `Comptroller CheckMembership Geoff cZRX`
* "Comptroller CheckListed <CToken>" - Returns true if market is listed, false otherwise.
  * E.g. "Comptroller CheckListed cZRX"

## CToken Values
* `CToken <CToken> UnderlyingBalance <User>` - Returns a user's underlying balance (based on given exchange rate)
  * E.g. `CToken cZRX UnderlyingBalance Geoff`
* `CToken <CToken> BorrowBalance <User>` - Returns a user's borrow balance (including interest)
  * E.g. `CToken cZRX BorrowBalance Geoff`
* `CToken <CToken> TotalBorrowBalance` - Returns the cToken's total borrow balance
  * E.g. `CToken cZRX TotalBorrowBalance`
* `CToken <CToken> Reserves` - Returns the cToken's total reserves
  * E.g. `CToken cZRX Reserves`
* `CToken <CToken> Comptroller` - Returns the cToken's comptroller
  * E.g. `CToken cZRX Comptroller`
* `CToken <CToken> PriceOracle` - Returns the cToken's price oracle
  * E.g. `CToken cZRX PriceOracle`
* `CToken <CToken> ExchangeRateStored` - Returns the cToken's exchange rate (based on balances stored)
  * E.g. `CToken cZRX ExchangeRateStored`
* `CToken <CToken> ExchangeRate` - Returns the cToken's current exchange rate
  * E.g. `CToken cZRX ExchangeRate`

## Erc-20 Values

* `Erc20 <Erc20> Address` - Returns address of ERC-20 contract
  * E.g. `Erc20 ZRX Address` - Returns ZRX's address
* `Erc20 <Erc20> Name` - Returns name of ERC-20 contract
  * E.g. `Erc20 ZRX Address` - Returns ZRX's name
* `Erc20 <Erc20> Symbol` - Returns symbol of ERC-20 contract
  * E.g. `Erc20 ZRX Symbol` - Returns ZRX's symbol
* `Erc20 <Erc20> Decimals` - Returns number of decimals in ERC-20 contract
  * E.g. `Erc20 ZRX Decimals` - Returns ZRX's decimals
* `Erc20 <Erc20> TotalSupply` - Returns the ERC-20 token's total supply
  * E.g. `Erc20 ZRX TotalSupply`
  * E.g. `Erc20 cZRX TotalSupply`
* `Erc20 <Erc20> TokenBalance <Address>` - Returns the ERC-20 token balance of a given address
  * E.g. `Erc20 ZRX TokenBalance Geoff` - Returns a user's ZRX balance
  * E.g. `Erc20 cZRX TokenBalance Geoff` - Returns a user's cZRX balance
  * E.g. `Erc20 ZRX TokenBalance cZRX` - Returns cZRX's ZRX balance
* `Erc20 <Erc20> Allowance owner:<Address> spender:<Address>` - Returns the ERC-20 allowance from owner to spender
  * E.g. `Erc20 ZRX Allowance Geoff Torrey` - Returns the ZRX allowance of Geoff to Torrey
  * E.g. `Erc20 cZRX Allowance Geoff Coburn` - Returns the cZRX allowance of Geoff to Coburn
  * E.g. `Erc20 ZRX Allowance Geoff cZRX` - Returns the ZRX allowance of Geoff to the cZRX cToken

## PriceOracle Values

* `Address` - Gets the address of the global price oracle
* `Price asset:<Address>` - Gets the price of the given asset

## Interest Rate Model Values

* `Address` - Gets the address of the global interest rate model

# Assertions

* `Equal given:<Value> expected:<Value>` - Asserts that given matches expected.
  * E.g. `Assert Equal (Exactly 0) Zero`
  * E.g. `Assert Equal (CToken cZRX TotalSupply) (Exactly 55)`
  * E.g. `Assert Equal (CToken cZRX Comptroller) (Comptroller Address)`
* `True given:<Value>` - Asserts that given is true.
  * E.g. `Assert True (Comptroller CheckMembership Geoff cETH)`
* `False given:<Value>` - Asserts that given is false.
  * E.g. `Assert False (Comptroller CheckMembership Geoff cETH)`
* `Failure error:<String> info:<String> detail:<Number?>` - Asserts that last transaction had a graceful failure with given error, info and detail.
  * E.g. `Assert Failure UNAUTHORIZED SUPPORT_MARKET_OWNER_CHECK`
  * E.g. `Assert Failure MATH_ERROR MINT_CALCULATE_BALANCE 5`
* `Revert` - Asserts that the last transaction reverted.
* `Success` - Asserts that the last transaction completed successfully (that is, did not revert nor emit graceful failure).
* `Log name:<String> ((key:<String> value:<Value>) ...)` - Asserts that last transaction emitted log with given name and key-value pairs.
  * E.g. `Assert Log Minted (("account" (User Geoff address)) ("amount" (Exactly 55)))`
