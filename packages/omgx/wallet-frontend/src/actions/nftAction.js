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
  return state.nft.list;
}

export function getNFTFactories () {
  const state = store.getState()
  return state.nft.factories;
}

export function getNFTContracts () {
  const state = store.getState()
  return state.nft.contracts;
}

export async function addNFT ( NFT ) {
  
  const state = store.getState();
  const UUID = NFT.UUID;

  //if we already have looked it up, no need to look up again. 
  if (state.nft.list[UUID]) {
    return state.nft.list[UUID];
  }
  
  const info = {
    UUID: NFT.UUID, 
    owner: NFT.owner, 
    url: NFT.url, 
    mintedTime: NFT.mintedTime, 
    decimals: 0,
    name:  NFT.name, 
    symbol:  NFT.symbol, 
    address: NFT.address,
    originID: NFT.originID,
    originAddress: NFT.originAddress,
    originChain: NFT.originChain,
    originFeeRecipient: NFT.originFeeRecipient,
    type: NFT.type
  }

  store.dispatch({
    type: 'NFT/GET/SUCCESS',
    payload: info
  })

  return info

}

export async function addNFTContract ( address ) {
  
  console.log("adding NFT Contract to state:", address)

  const state = store.getState();
    
  //if we already have looked it up, no need to look up again. 
  if (state.nft.contracts[address]) {
    return state.nft.contracts[address];
  }

  store.dispatch({
    type: 'NFT/ADDCONTRACT/SUCCESS',
    payload: address
  })

  return address

}

/* Allows one or more fields of an NFT factroy to be updated */

export async function changeNFTFactory ( Factory ) {
          
  const factory = {
    owner: Factory.owner, 
    address: Factory.address,
    mintedTime: Factory.mintedTime, 
    decimals: 0,
    symbol:  Factory.symbol, 
    layer: Factory.layer,
    name: Factory.name,
    originID: Factory.originID,
    originAddress: Factory.originAddress,
    originChain: Factory.originChain,
    originFeeRecipient: Factory.originFeeRecipient,
    haveRights: Factory.haveRights
  }

  store.dispatch({
    type: 'NFT/CREATEFACTORY/SUCCESS',
    payload: factory
  })

  return factory;

}

export async function addNFTFactory ( Factory ) {

  const state = store.getState();
    
  const address = Factory.address
    
  //if we already have looked it up, no need to look up again. 
  if (state.nft.factories[address]) {
    return state.nft.factories[address];
  }
  
  const factory = {
    owner: Factory.owner, 
    address,
    mintedTime: Factory.mintedTime, 
    decimals: 0,
    symbol:  Factory.symbol, 
    layer: Factory.layer,
    name: Factory.name,
    originID: Factory.originID,
    originAddress: Factory.originAddress,
    originChain: Factory.originChain,
    originFeeRecipient: Factory.originFeeRecipient,
    haveRights: Factory.haveRights
  }

  //console.log("nft factory:",factory)

  store.dispatch({
    type: 'NFT/CREATEFACTORY/SUCCESS',
    payload: factory
  })

  store.dispatch({
    type: 'NFT/ADDCONTRACT/SUCCESS',
    payload: address
  })

  return factory;

}