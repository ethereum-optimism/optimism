// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;

/**
 * @title ERC721Registry
 * 
 */
contract ERC721Registry {

    struct wallet {
        address[] addresses;
    }   
    
    mapping (address => wallet) locations;

    /**
     * @dev Emitted when a NFT contract addresss is added to a user's wallet.
     */
    event AddressRegistered(address indexed walletAddress, address indexed NFTContractAddress);

    constructor () {
    }

    function registerAddress(address walletAddress, address NFTContractAddress) public
    { 
        //note - we don't check for double adding...
        //less expensive to simply remove duplicates at the frontend, and of course,
        //any sane frontend will check first to make sure they are not wasting gas by 
        //registering their NFT contract address more than once per recipient
        //Also, we don't bother with removing, since there are better ways to do that - 
        //i.e. by burning the actual NFTs - stale addresses will just accumulate 
        locations[walletAddress].addresses.push(NFTContractAddress);  

        emit AddressRegistered(walletAddress, NFTContractAddress);   
    }  

    function lookupAddress(address walletAddress) public view returns(address[] memory) 
    { 
        return locations[walletAddress].addresses;     
    }  

}

