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
Returns Token info
If we don't have the info, try to get it
*/

export async function getToken ( tokenContractAddress ) {

  //this *might* be coming from a person, and or copy-paste from Etherscan
  //so need toLowerCase()
  /*****************************************************************/
  const _tokenContractAddress = tokenContractAddress.toLowerCase();
  /*****************************************************************/

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

/* 
Get the token info from networkService.web3.eth.Contract
*/
export async function addToken ( tokenContractAddress ) {

  const state = store.getState();

  //this *might* be coming from a person, and or copy-past from Etherscan
  //so need to toLowerCase()
  /*****************************************************************/
  const _tokenContractAddress = tokenContractAddress.toLowerCase();
  /*****************************************************************/
    
  //if we already have looked it up, no need to look up again. 
  if (state.tokenList[_tokenContractAddress]) {
    console.log("tokenAction = already in list:",_tokenContractAddress)
    return state.tokenList[_tokenContractAddress];
  }

  try {
    
    let tokenContract = null;
    if (networkService.L1orL2 === 'L1') {
      tokenContract = new networkService.l1Web3Provider.eth.Contract(erc20abi, _tokenContractAddress);
    } else {
      tokenContract = new networkService.l2Web3Provider.eth.Contract(erc20abi, _tokenContractAddress);
    }

    const [ _symbol, _decimals, _name ] = await Promise.all([
      tokenContract.methods.symbol().call(),
      tokenContract.methods.decimals().call(),
      tokenContract.methods.name().call()
    ]).catch(e => [ null, null, null ]);
    
    const decimals = _decimals ? Number(_decimals.toString()) : 'NOT ON ETHEREUM';
    const symbol = _symbol || 'NOT ON ETHEREUM';
    const name = _name || 'NOT ON ETHEREUM';

    const tokenInfo = {
      currency: _tokenContractAddress,
      symbol,
      decimals,
      name,
      redalert: _decimals ? false : true 
    };

    store.dispatch({
      type: 'TOKEN/GET/SUCCESS',
      payload: tokenInfo
    });

    return tokenInfo;

  } catch (error) {

    store.dispatch({
      type: 'TOKEN/GET/FAILURE',
      payload: {currency: _tokenContractAddress, symbol: 'Not found', error: 'Not found'},
    });

    return {currency: _tokenContractAddress, symbol: 'Not found', error: 'Not found'};
  }
}