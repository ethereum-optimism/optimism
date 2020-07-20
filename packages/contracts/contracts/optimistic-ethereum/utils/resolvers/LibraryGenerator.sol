pragma solidity ^0.5.0;

import { AddressResolver } from "./AddressResolver.sol";
import { EthMerkleTrie } from "../libraries/EthMerkleTrie.sol";
import { RollupMerkleUtils } from "../libraries/RollupMerkleUtils.sol";
import { RLPEncode } from "../libraries/RLPEncode.sol";
import { ContractAddressGenerator } from "../libraries/ContractAddressGenerator.sol";

contract LibraryGenerator {
    AddressResolver addressResolver;

    constructor(address _addressResolver) public {
        addressResolver = AddressResolver(_addressResolver);
    }

    function makeAll() public {
        makeEthMerkleTrie();
        makeRollupMerkleUtils();
        makeRLPEncode();
    }

    function makeEthMerkleTrie() public {
        EthMerkleTrie lib = new EthMerkleTrie();
        addressResolver.setAddress("EthMerkleTrie", address(lib));
    }

    function makeRollupMerkleUtils() public {
        RollupMerkleUtils lib = new RollupMerkleUtils();
        addressResolver.setAddress("RollupMerkleUtils", address(lib));
    }

    function makeRLPEncode() public {
        RLPEncode lib = new RLPEncode();
        addressResolver.setAddress("RLPEncode", address(lib));
    }

    function makeContractAddressGenerator() public {
        ContractAddressGenerator lib = new ContractAddressGenerator();
        addressResolver.setAddress("ContractAddressGenerator", address(lib));
    }
}