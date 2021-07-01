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

import store from 'store';

export function getNFTs () {
  const state = store.getState()
  return state.nftList;
}

export async function addNFT ( NFTproperties ) {

  const state = store.getState();
  const UUID = NFTproperties.UUID;
    
  //if we already have looked it up, no need to look up again. 
  if (state.nftList[UUID]) {
    return state.nftList[UUID];
  }
  
  const nftInfo = {
    UUID: NFTproperties.UUID, 
    owner: NFTproperties.owner, 
    url: NFTproperties.url, 
    mintedTime: NFTproperties.mintedTime, 
    decimals: 0,
    name:  NFTproperties.name, 
    symbol:  NFTproperties.symbol, 
  };

  //console.log("nftInfo0:",nftInfo)

  store.dispatch({
    type: 'NFT/GET/SUCCESS',
    payload: nftInfo
  })

  return nftInfo;

}