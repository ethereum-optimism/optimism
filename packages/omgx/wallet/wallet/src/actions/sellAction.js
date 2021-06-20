/*
  Varna - A Privacy-Preserving Marketplace
  Varna uses Fully Homomorphic Encryption to make markets fair. 
  Copyright (C) 2021 Enya Inc. Palo Alto, CA

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

import cryptoWorker from 'workerize-loader!../cryptoWorker/cryptoWorker'; // eslint-disable-line import/no-webpack-loader-syntax
import { encode, decode } from 'base64-arraybuffer';
import { ethers } from 'ethers';
import md5 from 'md5';
import BN from 'bn.js';
import store from 'store';
import { v4 as uuidv4 } from 'uuid';

import { openAlert, openError } from './uiAction';
import { AESEncrypt, AESDecrypt } from 'cryptoWorker/cryptoWorker';

import { powAmount } from 'util/amountConvert';
import { accSub, accMul } from 'util/calculation';

import networkService from 'services/networkService';

import { SELLER_OPTIMISM_API_URL } from 'Settings';

const encryptItemForSellerBegin = () => ({
  type: 'ENCRYPT_ITEM'
})

const encryptItemForSellerSuccess = () => ({
  type: 'ENCRYPT_ITEM_SUCCESS'
})

const encryptItemForSellerFailure = (data) => ({
  type: 'ENCRYPT_ITEM_FAILURE',
  payload: data,
})

const uploadItemFilesBegin = () => ({
  type: "UPLOAD_ITEM_FILES",
})

const uploadItemFilesSuccess = () => ({
  type: "UPLOAD_ITEM_FILES_SUCCESS",
})

const uploadItemFilesFailure = (data) => ({
  type: "UPLOAD_ITEM_FILES_FAILURE",
  payload: data,
})

const configureItemToOMGXBegin = () => ({
  type: "CONFIGURE_ITEM_TO_PLASMA",
})

const configureItemToOMGXSuccess = () => ({
  type: "CONFIGURE_ITEM_TO_PLASMA_SUCCESS",
})

const configureItemToOMGXFailure = (data) => ({
  type: "CONFIGURE_ITEM_TO_PLASMA_FAILURE",
  payload: data,
})

const isItemOpenOrClosedBegin = () => ({
  type: 'ITEM_OPEN_OR_CLOSED'
})

const isItemOpenOrClosedSuccess = (data) => ({
  type: 'ITEM_OPEN_OR_CLOSED_SUCCESS',
  payload: data,
})

const isItemOpenOrClosedFailure = (data) => ({
  type: 'ITEM_OPEN_OR_CLOSED_FAILURE',
  payload: data,
})

const downloadItemCiphertextBegin = (itemID) => ({
  type: 'DOWNLOAD_ITEM_CIPHERTEXT',
  payload: { itemID },
})

const downloadItemCiphertextSuccess = (itemID, error) => ({
  type: 'DOWNLOAD_ITEM_CIPHERTEXT_SUCCESS',
  payload: { itemID, error },
})

const downloadItemCiphertextFailure = (itemID, ciphertext) => ({
  type: 'DOWNLOAD_ITEM_CIPHERTEXT_FAILURE',
  payload: { itemID, ciphertext },
})

/****************************/
/* Decrypt ask by using FHE */
/****************************/
const decryptItemBegin = (itemID) => ({
  type: 'DECRYPT_ITEM',
  payload: { itemID },
})

const decryptItemSuccess = (itemID, cleartext) => ({
  type: 'DECRYPT_ITEM_SUCCESS',
  payload: { itemID, cleartext }
})

const decryptItemFailure = (itemID, error) => ({
  type: 'DECRYPT_ITEM_FAILURE',
  payload: { itemID, error }
})
/****************************/

/****************************/
/* Decypt ask by using AES */
/****************************/
const decryptItemCacheBegin = (itemID) => ({
  type: 'DECRYPT_ITEM_CACHE',
  payload: { itemID },
})

const decryptItemCacheSuccess = (itemID, cleartext) => ({
  type: 'DECRYPT_ITEM_CACHE_SUCCESS',
  payload: { itemID, cleartext }
})

const decryptItemCacheFailure = (itemID, error) => ({
  type: 'DECRYPT_ITEM_CACHE_FAILURE',
  payload: { itemID, error }
})
/****************************/

const deleteItemBegin = (itemID) => ({
  type: 'DELETE_ITEM',
  payload: { itemID },
})

const deleteItemSuccess = (itemID) => ({
  type: 'DELETE_ITEM_SUCCESS',
  payload: { itemID }
})

const deleteItemFailure = (itemID, error) => ({
  type: 'DELETE_ITEM_FAILURE',
  payload: { itemID, error },
})

const acceptBidBegin = (itemID, bidID) => ({
  type: 'ACCEPT_BID',
  payload: { itemID, bidID },
})

const acceptBidSuccess = (itemID, bidID) => ({
  type: 'ACCEPT_BID_SUCCESS',
  payload: { itemID, bidID }
})

const acceptBidFailure = (itemID, bidID, error) => ({
  type: 'ACCEPT_BID_FAILURE',
  payload: { itemID, bidID, error },
})

/****************************/
/* Seller Accepts Bid data */
/****************************/
const getSellerAcceptBidDataBegin = () => ({
  type: 'GET_SELLER_ACCEPT_BID_DATA',
})

const getSellerAcceptBidDataSuccess = (data) => ({
  type: 'GET_SELLER_ACCEPT_BID_DATA_SUCCESS',
  payload: data,
})

const getSellerAcceptBidDataFailure = (error) => ({
  type: 'GET_SELLER_ACCEPT_BID_DATA_FAILURE',
  payload: error,
})
/****************************/

const uploadItemFiles = (message, itemID, itemToSend, itemToReceive, address) => (dispatch) => {
  dispatch(uploadItemFilesBegin());
      
  return fetch(SELLER_OPTIMISM_API_URL + "list.item", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      publicKey: message.data.fhePublicKey,
      multiKey: message.data.fheMultiKey,
      rotaKey: message.data.fheRotaKey,
      ciphertext: message.data.fheCiphertext,
      itemID: itemID,
      symbolA: itemToSend.symbol,
      symbolB: itemToReceive.symbol,
      address,
    }),
  }).then(res => {
    if (res.status === 201) {
      dispatch(uploadItemFilesSuccess());
      return { status: 201 }
    } else {
      dispatch(uploadItemFilesFailure(res.status));
      return { status: res.status }
    }
  })
}

const configureItemToOMGX = (itemID, itemToSend, itemToReceive, address) => async (dispatch) => {
  dispatch(configureItemToOMGXBegin());
  /************************************************/
  /*********  Removed the smart contract **********/
  /************************************************/
  try {
    // // fire the smart contract
    // const tx = await networkService.VarnaPoolContract.listItem(
    //   address,
    //   itemID,
    //   `${itemToSend.symbol}-${itemToReceive.symbol}/${itemID}`,
    //   new Date().getTime(),
    // );

    // if (tx) {
      dispatch(configureItemToOMGXSuccess());
      dispatch(openAlert("Item was successfully encrypted and dispatched. Now writing to the smart contract"));
    // } else {
    //   dispatch(openError("Failed to write to Plasma"));
    //   dispatch(configureItemToOMGXFailure(404));
    // }
  } catch(error) {
    dispatch(openError("Unknown error"));
    dispatch(configureItemToOMGXFailure(404));
  }
}

/* List an item */
export const listItem = (
  itemToSend, 
  itemToSendAmount, 
  itemToReceive, 
  sellerExchangeRate, 
  FHEseed, 
  ) => (dispatch) => {
  console.log("listBid: Starting the item listing process")
  var cryptoWorkerThreadID = crypto.getRandomValues(new Uint32Array(1)).toString(16);
  
  dispatch(encryptItemForSellerBegin());

  const workerInstance = cryptoWorker();

  workerInstance.generateItem(
    itemToSend, 
    itemToSendAmount, 
    itemToSendAmount,/*this is not a bug - needed to update remaining amount */
    itemToReceive, 
    sellerExchangeRate, 
    FHEseed, 
    cryptoWorkerThreadID
  );

  workerInstance.addEventListener('message', (message) => {
    if (message.data.status === "success" && 
        message.data.type === "listItem" && 
        message.data.cryptoWorkerThreadID === cryptoWorkerThreadID
    ) {
      dispatch(encryptItemForSellerSuccess());

      // Generate hashed itemID
      const itemID = md5(JSON.stringify(message.data.fheCiphertext));
      // Generate hashed address
      const address = md5(networkService.account);

      dispatch(uploadItemFiles(message, itemID, itemToSend, itemToReceive, address)).then(statusCode => {
        if (statusCode.status === 201) {
          dispatch(configureItemToOMGX(itemID, itemToSend, itemToReceive, address));
        } else {
          dispatch(uploadItemFilesFailure(statusCode.status));
          dispatch(openError("Failed to broadcast your listing"));
        }
      })

    } else if (message.data.status === "failure") {
      dispatch(encryptItemForSellerFailure(404));
      dispatch(openError("Failed to encrypt your item"));
    }
  });
}

export const isItemOpenOrClosed = () => (dispatch) => {

  dispatch(isItemOpenOrClosedBegin());

  // Generate hashed address
  const address = md5(networkService.account);

  return fetch(SELLER_OPTIMISM_API_URL + "download.item.status", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ address }),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      dispatch(isItemOpenOrClosedFailure(res.status));
      return ""
    }
  }).then(data => {
    if (data !== "") {
      dispatch(isItemOpenOrClosedSuccess(data.data));
      return data.data;
    }
    return "";
  })
}

export const downloadItemCiphertext = (itemID) => (dispatch) => {

  dispatch(downloadItemCiphertextBegin(itemID));

  return fetch(SELLER_OPTIMISM_API_URL + "download.item.ciphertext", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      itemID
    }),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      dispatch(downloadItemCiphertextFailure(itemID, res.status));
      return ""
    }
  }).then(data => {
    if (data !== "") {
      dispatch(downloadItemCiphertextSuccess(itemID, data.ciphertext));
      return data.ciphertext;
    } else {
      return ""
    }
  })
}

export const decryptItem = (itemID, FHEseed, AESKey, ciphertext) => (dispatch) => {
  dispatch(decryptItemBegin(itemID));

  const workerInstance = cryptoWorker();
  workerInstance.decryptItem(itemID, ciphertext, FHEseed);

  workerInstance.addEventListener('message', (message) => {
    if (message.data.status === "success" && 
        message.data.type === "decryptAsk" &&
        message.data.itemID === itemID
    ) {
        const itemCleartext = message.data.itemCleartext.slice(0, 11);
        //console.log({itemCleartext});
        dispatch(decryptItemSuccess(itemID, itemCleartext));
        // AES encrypt data
        AESEncrypt(JSON.stringify(itemCleartext), AESKey).then(ciphertext => {
          const bufferCiphertext = ciphertext.ciphertext;
          const bufferIV = ciphertext.iv;
          const base64Ciphertext = {
            ciphertext: encode(bufferCiphertext),
            iv: encode(bufferIV),
          }

          // store ciphertext
          let decryptedItem = localStorage.getItem("decryptedItem");
          if (!decryptedItem) {
            decryptedItem = {[itemID]: base64Ciphertext}
          } else {
            decryptedItem = JSON.parse(decryptedItem);
            decryptedItem = {
              ...decryptedItem,
              [itemID]: base64Ciphertext,
            }
          }
          localStorage.setItem("decryptedItem", JSON.stringify(decryptedItem));
        })
    } else if (
      message.data.status === "failure" && 
      message.data.type === "decryptAsk" &&
      message.data.itemID === itemID
    ) {
      dispatch(decryptItemFailure(itemID, 404));
    }
  });

}

export const loadItem = (itemID, FHEseed, AESKey) => (dispatch) => {
  dispatch(downloadItemCiphertext(itemID)).then(ciphertext => {
    dispatch(decryptItem(itemID, FHEseed, AESKey, ciphertext));
  })
}

export const findNextDecryptBiditemIDIndex = (itemOpenList, latestOpenitemID) => {

  let decryptedItemCache = localStorage.getItem("decryptedItem");

  let decrypteditemIDCache = null;

  if (decryptedItemCache) {
    decrypteditemIDCache = Object.keys(JSON.parse(decryptedItemCache));
  }

  let nextDecryptBiditemIDIndex = itemOpenList.indexOf(latestOpenitemID) < itemOpenList.length - 1 ? itemOpenList.indexOf(latestOpenitemID) : null;

  // check if the cache has the ask data
  while (nextDecryptBiditemIDIndex < itemOpenList.length && nextDecryptBiditemIDIndex !== null) {
    if (decrypteditemIDCache) {
      if (decrypteditemIDCache.includes(itemOpenList[nextDecryptBiditemIDIndex])) {
        nextDecryptBiditemIDIndex += 1;
      } else {
        nextDecryptBiditemIDIndex += 1;
        break;
      }
    } else {
      nextDecryptBiditemIDIndex += 1;
      break;
    }
  }

  return nextDecryptBiditemIDIndex;
}


export const deleteItem = (itemID) => (dispatch) => {

  dispatch(deleteItemBegin(itemID));

  // Generate hashed address
  const address = md5(networkService.account);

  return fetch(SELLER_OPTIMISM_API_URL + "delete.item", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      itemID, address
    }),
  }).then(res => {
    if (res.status === 201) {
      dispatch(openAlert("Your item listing was deleted"));
      dispatch(deleteItemSuccess(itemID));
    } else {
      dispatch(openError("Failed to delete your item"));
      dispatch(deleteItemFailure(itemID, res.status));
    }
  })
}

export const acceptBid = ( cMD ) => async (dispatch) => {

  /*
    itemID, 
    bidID, 
    address, 
    sellerItemToSend,
    sellerItemToReceive,
    sellerItemToSendAmount, 
    sellerItemToSendAmountRemain,
    sellerExchangeRate,
    agreeAmount, 
    agreeExchangeRate, 
    FHEseed, 
  */

  const state = store.getState();
  const balance = state.balance;
  const itemOpenList = state.sell.itemOpenList;

  const availableBalanceArray = balance.childchain.filter(i => i.currency === cMD.sellerItemToSend.currency);
  let availableBalance = 0;
  if (availableBalanceArray.length) {
    availableBalance = availableBalanceArray[0].amount;
  }

  const tokenList = state.tokenList;
  // Security issue!!
  const sellerItemToSendDecimals = tokenList[cMD.sellerItemToSend.currency.toLowerCase()].decimals;
  const sellerItemToSendAgreeAmount = powAmount(cMD.agreeAmount, sellerItemToSendDecimals);
  // Security issue!!
  const sellerItemToReceiveDecimals = tokenList[cMD.sellerItemToReceive.currency.toLowerCase()].decimals;
  const sellerItemToReceiveAmount = powAmount(accMul(cMD.agreeAmount, cMD.agreeExchangeRate), sellerItemToReceiveDecimals);

  if (availableBalance.lt(new BN(sellerItemToSendAgreeAmount))) {
    dispatch(openError(`You don't have enough ${cMD.sellerItemToSend.symbol} to cover the amount you want to send.`));
    dispatch(acceptBidFailure(cMD.itemID, cMD.bidID, 404));
  } else {
    // The remaining amount of items that seller wants to send
    const sellerItemToSendAmountRemainUpdated = accSub(cMD.sellerItemToSendAmountRemain, cMD.agreeAmount);

    dispatch(acceptBidBegin(cMD.itemID, cMD.bidID));

    let cryptoWorkerThreadID = "";
    while (cryptoWorkerThreadID.length < 20) cryptoWorkerThreadID += Math.random().toString(36).substr(2);
  
    // Web worker
    const workerInstance = cryptoWorker();
    workerInstance.generateItem(
      cMD.sellerItemToSend, 
      cMD.sellerItemToSendAmount, 
      sellerItemToSendAmountRemainUpdated, 
      cMD.sellerItemToReceive, 
      cMD.sellerExchangeRate,
      cMD.FHEseed, 
      cryptoWorkerThreadID,
    );
      
    workerInstance.addEventListener('message', async (message) => {
      if (message.data.status === "success" && 
          message.data.type === "listItem" && 
          message.data.cryptoWorkerThreadID === cryptoWorkerThreadID
      ) {
        // upload the updated ciphertext to S3
        const uploadFilesStatus = await dispatch(uploadItemFiles(
          message, 
          cMD.itemID, 
          cMD.sellerItemToSend, 
          cMD.sellerItemToReceive,
          md5(networkService.account)
        ));
        
        if (uploadFilesStatus.status === 201) {
          // update the data in cache
          dispatch(decryptItem(cMD.itemID, cMD.FHEseed, cMD.AESKey, message.data.fheCiphertext));

          try {
            const UUID = uuidv4();
            const swapID = ethers.utils.soliditySha3(UUID);
            const openValue = sellerItemToSendAgreeAmount;
            const openContractAddress = cMD.sellerItemToSend.currency;
            const closeValue = sellerItemToReceiveAmount;
            const closeTrader = cMD.address;
            const closeContractAddress = cMD.sellerItemToReceive.currency;
            
            const swapStatus = await networkService.AtomicSwapContract.open(
              swapID,
              openValue,
              openContractAddress,
              closeValue,
              closeTrader,
              closeContractAddress,
            );
            const swapRes = await swapStatus.wait();
            
            if (swapRes) {
              const uploadSwapBody = await uploadSellerAcceptBidData({
                UUID,
                itemID: cMD.itemID,
                bidID: cMD.bidID,
                sellerAddress: networkService.account,
                buyerAddress: closeTrader,
                agreeAmount: cMD.agreeAmount,
                agreeExchangeRate: cMD.agreeExchangeRate,
                currencyA: cMD.sellerItemToSend.currency,
                currencyB: cMD.sellerItemToReceive.currency,
                symbolA: cMD.sellerItemToSend.symbol,
                symbolB: cMD.sellerItemToReceive.symbol
              })
  
              if (uploadSwapBody === 201) {
                dispatch(acceptBidSuccess(cMD.itemID, cMD.bidID));
                dispatch(openAlert("Swap was sent"));
                dispatch(getSellerAcceptBidData(itemOpenList));
              } else {
                dispatch(acceptBidFailure(cMD.itemID, cMD.bidID, 404));
                dispatch(openError("Unknown error"));
              }
            } else {
              dispatch(acceptBidFailure(cMD.itemID, cMD.bidID, 404));
              dispatch(openAlert("Failed to swap"));
            }

          } catch(error) {
            if (error.message.includes("User denied message signature.")) {
              dispatch(acceptBidFailure(cMD.itemID, cMD.bidID, 404));
              dispatch(openAlert("Cancelled the signature"));
            } else {
              dispatch(acceptBidFailure(cMD.itemID, cMD.bidID, 404));
              dispatch(openError("Could not sign the transaction. Please try again."));
            }
          }

        } else {
          dispatch(acceptBidFailure(cMD.itemID, cMD.bidID, 404));
          dispatch(openError("Failed to upload your file"));
        }

      }
    })

  }
}

export const decryptItemCache = (decryptedItem, decryptedItemCache, decryptedItemCacheError, itemOpenList, AESKey) => async (dispatch) => {
  for (let eachitemID of itemOpenList) {
    // check whether itemID is decrypted  check whether itemID is in cache      check whether cached ask is decrypted
    if (!decryptedItem[eachitemID] && decryptedItemCache[eachitemID] && typeof decryptedItemCacheError[eachitemID] === 'undefined') {
      dispatch(decryptItemCacheBegin(eachitemID));

      const base64Ciphertext = decryptedItemCache[eachitemID].ciphertext;
      const base64IV = decryptedItemCache[eachitemID].iv;
      const ciphertText = {
        ciphertext: decode(base64Ciphertext),
        iv: decode(base64IV),
      }

      try{
        const cleartextString = await AESDecrypt(ciphertText, AESKey);
        const cleartext = JSON.parse(cleartextString);
        if (Array.isArray(cleartext)) {
          dispatch(decryptItemCacheSuccess(eachitemID, cleartext));
        } else {
          dispatch(decryptItemCacheFailure(eachitemID, 'Error'));
        }
      } catch (error) {
        dispatch(decryptItemCacheFailure(eachitemID, 'Error'));
      }

    }
  }

}

export const getSellerAcceptBidData = (itemIDList) => (dispatch) => {
  dispatch(getSellerAcceptBidDataBegin());

  return fetch(SELLER_OPTIMISM_API_URL + "download.agreement", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      itemIDList, address: networkService.account,
    }),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      dispatch(getSellerAcceptBidDataFailure(res.status));
      return ""
    }
  }).then(data => {
    if (data !== "") {
      dispatch(getSellerAcceptBidDataSuccess(data));
    }
  })
}

const uploadSellerAcceptBidData = (cMD) => {
  return fetch(SELLER_OPTIMISM_API_URL + "upload.agreement", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(cMD),
  }).then(res => {
    return res.status
  })
}