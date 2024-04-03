---
description: Hybrid Compute Example - Minting NFTs with Random Attributes
---

<figure><img src="../../../assets/hc-under-upgrade.png" alt=""><figcaption></figcaption></figure>

# Monster Minting

<figure><img src="../../../assets/mint your monster.png" alt=""><figcaption></figcaption></figure>

Clone the repository, open it, and install packages with `yarn`:

```bash
$ git clone https://github.com/bobanetwork/boba.git
$ cd boba
$ yarn
$ yarn build
```

As for every chain, you need an account with some ETH (to deploy contracts) and since you will be using Turing, you also need some BOBA in that same account. In the deploy script (`./boba_community/turing-monsters/test/NFTMonsterV2.ts`), specify your private key or set the environment variable `PRIVATE_KEY=0x..` when running the script.

```javascript

// uncomment the correct addresses

// Rinkeby
// const BOBAL2Address = '0xF5B97a4860c1D81A1e915C40EcCB5E4a5E6b8309'
// const BobaTuringCreditRinkebyAddress = '0x208c3CE906cd85362bd29467819d3AcbE5FC1614'

// for example if you are on Mainnet-Test, uncomment these

// Mainnet-Test
const BOBAL2Address = '0x58597818d1B85EF96383884951E846e9D6D03956'
const BobaTuringCreditRinkebyAddress = '0xE654ba86Ea0B59a6836f86Ec806bfC9449D0aD0A'

// provide your PK here
const testPrivateKey = process.env.PRIVATE_KEY ?? '0x____'

```

You can also do this via a hardware wallet, a mnemonic, via `hardhat.config.js`, or whatever you usually do. Whatever account/key you use, it needs some ETH and BOBA - small amounts should be sufficient.

<figure><img src="../../../assets/getting geth eth and geth boba.png" alt=""><figcaption></figcaption></figure>

If you do not have testnet ETH, you can get some here [Faucets](https://docs.boba.network/developer/faucets).

<figure><img src="../../../assets/testing the turing monster nft.png" alt=""><figcaption></figcaption></figure>

To run the tests you will also need some Goerli ETH on Goerli (L1) as the tests also test the NFT bridging functionality.

```bash
$ cd boba_community/hc-monsters
$ yarn build
$ PRIVATE_KEY=0x... yarn test:rinkeby # for testing on rinkeby, for example

# other choices are local and mainnet
```

Ok, all done. Enjoy. The terminal will give you all the information you need to mint and send a Turing monster to your friends:

```
  Turing NFT Random 256
    Turing Helper contract deployed at 0x3a622DB2db50f463dF562Dc5F341545A64C580fc
    ERC721 contract deployed at 0x6A47346e722937B60Df7a1149168c0E76DD6520f
    adding your ERC721 as PermittedCaller to TuringHelper 0x0000000000000000000000006a47346e722937b60df7a1149168c0e76dd6520f
    Credit Prebalance 0
    BOBA Balance in your account 140000000000000000000
    ✓ Should register and fund your Turing helper contract in turingCredit (196ms)
    ERC721 contract whitelisted in TuringHelper (1 = yes)? 1
    ✓ Your ERC721 contract should be whitelisted (59ms)
    ✓ should mint an NFT with random attributes (104ms)
    Turing URI = data:application/json;base64,eyJuYW1lIjogIlR1cmluZ01vbnN0ZXIiLCAiZGVzY3JpcHRpb24iOiAiQm9vb29Ib29vbyIsICJpbWFnIn0=
    ✓ should get an svg

```

To deploy run:

```bash
$ cd boba_community/hc-monsters
$ yarn build
$ PRIVATE_KEY=0x... yarn run deploy -- --network boba_rinkeby
```

Add the ERC721 as permitted caller to the deployed TuringHelper. Call the method `startTrading()` once you feel ready so that your community is able to mint their NFTs.

<figure><img src="../../../assets/solidity code walkthrough.png" alt=""><figcaption></figcaption></figure>

The ERC721 contract is largely standard, except for needing to provide the address of the `TuringHelper` contract. Nevertheless, the contract has been distributed into several smaller contracts to make them easily reusable for your own project.

Core features:

* You'll mint a random tokenID issued by Turing; `@ref RandomlyAssigned.sol:nextToken()`
* MetaData is onChain and also is randomized via Turing; `@ref WithOnChainMetaData.sol:getMetadata()`
* Recover functions for tokens when someone accidentally sends funds to the contract; `@ref WithRecover.sol`
* Max Mint per wallet is limited and minting a NFT costs an additional fee (see the PRICE constant); `@ref NFTMonsterV2.sol:mint()`
* NFT implements the `IERC2981` standard for royalty fees; `@ref NFTMonsterV2.sol:royaltyInfo()`
* Minting revenue will be split across project owners via claim function; `@ref NFTMonsterV2.sol:withdraw()`
* NFT not tradeable until project owner calls `startTrading()`; `@ref NFTMonsterV2.sol:saleIsOpen[modifier]`

```javascript

  function mint(uint256 _count) external payable saleIsOpen {
    uint256 total = tokenCount();
    require(_count > 0, "Mint more than 0");
    require(total + _count <= totalSupply(), "Max limit");
    require(msg.value >= price(_count), "Value below price");

    amountMintedInPublicSale[_msgSender()] = amountMintedInPublicSale[_msgSender()] + _count;
    require(amountMintedInPublicSale[_msgSender()] <= MAX_MINT_IN_PUBLIC);

    for (uint256 i = 0; i < _count; i++) {
      _mintSingle(_msgSender());
    }
  }

    function getSVG(uint tokenId) private view returns (string memory) {

        require(_exists(tokenId), "ERC721getSVG: URI get of nonexistent token");

        string memory genome = _tokenURIs[tokenId];
        bytes memory i_bytes = abi.encodePacked(genome); // get the bytes

        uint8 attribute_a = uint8(i_bytes[0]); // peel off the first byte (0-255)
        uint8 attribute_b = uint8(i_bytes[1]);
        // uint8 attribute_c = uint8(i_bytes[2]);
        // ...
  ...
        string[4] memory part;

        string memory colorEye = "C15AA2";
        if(attribute_a > 128){
          colorEye = "54B948";
        }
  ...

        part[0] = "<svg x='0px' y='0px' viewBox='0 0 300 300' style='enable-background:new 0 0 300 300;' xml:space='preserve'><style type='text/css'>.st0{fill:#";
  ...
        return string(abi.encodePacked(part[0], colorEye, part[1], colorBody, part[2], part[3]));
    }
```
