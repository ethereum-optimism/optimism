/*
Copyright 2019-present OmiseGO Pte Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. */

import erc20abi from 'human-standard-token-abi';
import networkService from 'services/networkService';
import store from 'store';

/*
import { getNFTs, addNFT } from 'actions/nftAction';
*/


/* 
Returns Token info
If we don't have the info, try to get it
*/
/*
export async function getToken ( tokenContractAddress ) {

  //this *might* be coming from a person, and or copy-paste from Etherscan
  //so need toLowerCase()
  // *****************************************************************
  const _tokenContractAddress = tokenContractAddress.toLowerCase();
  // *****************************************************************

  const state = store.getState();
  if (state.tokenList[_tokenContractAddress]) {
    //console.log("tokenAction = token in list:",_tokenContractAddress)
    return state.tokenList[_tokenContractAddress];
  } else {
    console.log("tokenAction = token not yet in list:",_tokenContractAddress)
    const tokenInfo = await addToken( _tokenContractAddress )
    return tokenInfo;
  }
}
*/

export async function getNFTs () {
  const state = store.getState()
  //console.log("export async function getNFTs")
  //console.log(state.nftList)
  return state.nftList;
}

/* 
Get the token info from networkService.web3.eth.Contract
*/
export async function addNFT ( NFTproperties ) {

  const state = store.getState();
  const UUID = NFTproperties.UUID;
    
  //if we already have looked it up, no need to look up again. 
  if (state.nftList[UUID]) {
    console.log("nftAction - already in list:",UUID)
    console.log(state.nftList[UUID])
    return state.nftList[UUID];
  }
  
  const nftInfo = {
    //currency: _tokenContractAddress,
    //symbol,
    //name,
    //redalert: _decimals ? false : true 
    UUID: NFTproperties.UUID, 
    owner: NFTproperties.owner, 
    url: NFTproperties.url, 
    mintedTime: NFTproperties.mintedTime, 
    decimals: 0,
  };

  store.dispatch({
    type: 'NFT/GET/SUCCESS',
    payload: nftInfo
  })

  return nftInfo;

}