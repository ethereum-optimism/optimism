
import {Event} from '../Event';
import {addAction, World} from '../World';
import {Erc20} from '../Contract/Erc20';
import {Invokation, invoke} from '../Invokation';
import {
  getAddressV,
  getCoreValue,
  getNumberV,
  getStringV
} from '../CoreValue';
import {
  AddressV,
  NumberV,
  StringV,
  Value
} from '../Value';
import {Arg, Fetcher, getFetcherValue} from '../Command';
import {storeAndSaveContract} from '../Networks';
import {getContract, getTestContract} from '../Contract';
import {encodeABI} from '../Utils';

const ExistingToken = getContract("EIP20Interface");
const TetherInterface = getContract("TetherInterface");

const FaucetTokenHarness = getContract("FaucetToken");
const FaucetTokenNonStandardHarness = getContract("FaucetNonStandardToken");
const FaucetTokenReEntrantHarness = getContract("FaucetTokenReEntrantHarness");
const EvilTokenHarness = getContract("EvilToken");
const WBTCTokenHarness = getContract("WBTCToken");
const FeeTokenHarness = getContract("FeeToken");

export interface TokenData {
  invokation: Invokation<Erc20>,
  description: string,
  name: string,
  symbol: string,
  decimals?: number,
  address?: string,
  contract: string
}

export async function buildErc20(world: World, from: string, event: Event): Promise<{ world: World, erc20: Erc20, tokenData: TokenData }> {
  const fetchers = [
    new Fetcher<{ symbol: StringV, address: AddressV, name: StringV }, TokenData>(`
        #### Existing

        * "Existing symbol:<String> address:<Address> name:<String>" - Wrap an existing Erc20 token
          * E.g. "Erc20 Deploy Existing DAI 0x123...
      `,
      "Existing",
      [
        new Arg("symbol", getStringV),
        new Arg("address", getAddressV),
        new Arg("name", getStringV, { default: undefined }),
      ],
      async (world, { symbol, name, address }) => {
        const existingToken = await ExistingToken.at<Erc20>(world, address.val);
        const tokenName = name.val === undefined ? symbol.val : name.val;
        const decimals = await existingToken.methods.decimals().call();

        return {
          invokation: new Invokation<Erc20>(existingToken, null, null, null),
          description: "Existing",
          decimals: Number(decimals),
          name: tokenName,
          symbol: symbol.val,
          contract: 'ExistingToken'
        };
      }
    ),

    new Fetcher<{symbol: StringV, address: AddressV}, TokenData>(`
        #### ExistingTether

        * "Existing symbol:<String> address:<Address>" - Wrap an existing Erc20 token
          * E.g. "Erc20 Deploy ExistingTether USDT 0x123...
      `,
      "ExistingTether",
      [
        new Arg("symbol", getStringV),
        new Arg("address", getAddressV)
      ],
      async (world, {symbol, address}) => {
        return {
          invokation: new Invokation<Erc20>(await TetherInterface.at<Erc20>(world, address.val), null, null, null),
          description: "ExistingTether",
          name: symbol.val,
          symbol: symbol.val,
          contract: 'TetherInterface'
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV}, TokenData>(`
        #### NonStandard

        * "NonStandard symbol:<String> name:<String> decimals:<Number=18>" - A non-standard token, like BAT
          * E.g. "Erc20 Deploy NonStandard BAT \"Basic Attention Token\" 18"
      `,
      "NonStandard",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("decimals", getNumberV, {default: new NumberV(18)}),
      ],
      async (world, {symbol, name, decimals}) => {
        return {
          invokation: await FaucetTokenNonStandardHarness.deploy<Erc20>(world, from, [0, name.val, decimals.val, symbol.val]),
          description: "NonStandard",
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          contract: 'FaucetNonStandardToken'
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, fun:StringV, reEntryFunSig: StringV, reEntryFunArgs: StringV[]}, TokenData>(`
        #### ReEntrant

        * "ReEntrant symbol:<String> name:string fun:<String> funSig:<String> ...funArgs:<Value>" - A token that loves to call back to spook its caller
          * E.g. "Erc20 Deploy ReEntrant PHREAK PHREAK "transfer" "mint(uint256)" 0 - A token that will call back to a CToken's mint function

        Note: valid functions: totalSupply, balanceOf, transfer, transferFrom, approve, allowance
      `,
      "ReEntrant",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("fun", getStringV),
        new Arg("reEntryFunSig", getStringV),
        new Arg("reEntryFunArgs", getStringV, {variadic: true, mapped: true})
      ],
      async (world, {symbol, name, fun, reEntryFunSig, reEntryFunArgs}) => {
        const fnData = encodeABI(world, reEntryFunSig.val, reEntryFunArgs.map((a) => a.val));

        return {
          invokation: await FaucetTokenReEntrantHarness.deploy<Erc20>(world, from, [0, name.val, 18, symbol.val, fnData, fun.val]),
          description: "ReEntrant",
          name: name.val,
          symbol: symbol.val,
          decimals: 18,
          contract: 'FaucetTokenReEntrantHarness'
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV}, TokenData>(`
        #### Evil

        * "Evil symbol:<String> name:<String> decimals:<Number>" - A less vanilla ERC-20 contract that fails transfers
          * E.g. "Erc20 Deploy Evil BAT \"Basic Attention Token\" 18"
      `,
      "Evil",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("decimals", getNumberV, {default: new NumberV(18)})
      ],
      async (world, {symbol, name, decimals}) => {
        return {
          invokation: await EvilTokenHarness.deploy<Erc20>(world, from, [0, name.val, decimals.val, symbol.val]),
          description: "Evil",
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          contract: 'EvilToken'
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV}, TokenData>(`
        #### Standard

        * "Standard symbol:<String> name:<String> decimals:<Number>" - A vanilla ERC-20 contract
          * E.g. "Erc20 Deploy Standard BAT \"Basic Attention Token\" 18"
      `,
      "Standard",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("decimals", getNumberV, {default: new NumberV(18)})
      ],
      async (world, {symbol, name, decimals}) => {
        return {
          invokation: await FaucetTokenHarness.deploy<Erc20>(world, from, [0, name.val, decimals.val, symbol.val]),
          description: "Standard",
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          contract: 'FaucetToken'
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV}, TokenData>(`
        #### WBTC

        * "WBTC symbol:<String> name:<String>" - The WBTC contract
          * E.g. "Erc20 Deploy WBTC WBTC \"Wrapped Bitcoin\""
      `,
      "WBTC",
      [
        new Arg("symbol", getStringV, {default: new StringV("WBTC")}),
        new Arg("name", getStringV, {default: new StringV("Wrapped Bitcoin")})
      ],
      async (world, {symbol, name}) => {
        let decimals = 8;

        return {
          invokation: await WBTCTokenHarness.deploy<Erc20>(world, from, []),
          description: "WBTC",
          name: name.val,
          symbol: symbol.val,
          decimals: decimals,
          contract: 'WBTCToken'
        };
      }
    ),

    new Fetcher<{symbol: StringV, name: StringV, decimals: NumberV, basisPointFee: NumberV, owner: AddressV}, TokenData>(`
        #### Fee

        * "Fee symbol:<String> name:<String> decimals:<Number> basisPointFee:<Number> owner:<Address>" - An ERC20 whose owner takes a fee on transfers. Used for mocking USDT.
          * E.g. "Erc20 Deploy Fee USDT USDT 100 Root"
      `,
      "Fee",
      [
        new Arg("symbol", getStringV),
        new Arg("name", getStringV),
        new Arg("decimals", getNumberV),
        new Arg("basisPointFee", getNumberV),
        new Arg("owner", getAddressV)
      ],
      async (world, {symbol, name, decimals, basisPointFee, owner}) => {
        return {
          invokation: await FeeTokenHarness.deploy<Erc20>(world, from, [0, name.val, decimals.val, symbol.val, basisPointFee.val, owner.val]),
          description: "Fee",
          name: name.val,
          symbol: symbol.val,
          decimals: decimals.toNumber(),
          owner: owner.val,
          contract: 'FeeToken'
        };
      }
    ),
  ];

  let tokenData = await getFetcherValue<any, TokenData>("DeployErc20", fetchers, world, event);
  let invokation = tokenData.invokation;
  delete tokenData.invokation;

  if (invokation.error) {
    throw invokation.error;
  }
  const erc20 = invokation.value!;
  tokenData.address = erc20._address;

  world = await storeAndSaveContract(
    world,
    erc20,
    tokenData.symbol,
    invokation,
    [
      { index: ['Tokens', tokenData.symbol], data: tokenData }
    ]
  );

  return {world, erc20, tokenData};
}
