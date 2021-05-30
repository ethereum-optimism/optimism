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
import { orderBy } from 'lodash';
import { encode, decode } from 'base64-arraybuffer';
import oracleService from 'services/oracleService';

import { accDiv, accMul } from 'util/calculation';
import { arrayToCoin } from 'util/coinConvert';
import { AESEncrypt, AESDecrypt } from 'cryptoWorker/cryptoWorker';

import { SELLER_OPTIMISM_API_URL } from '../Settings';

/****************************/
/* Decrypt ask by using FHE */
/****************************/
const downloadBidOfferBegin = (bidID) => ({
  type: 'DOWNLOAD_BID_OFFER',
  payload: { bidID },
})

const downloadBidOfferSuccess = (bidID, file) => ({
  type: 'DOWNLOAD_BID_OFFER_SUCCESS',
  payload: { bidID, file },
})

const downloadBidOfferFailure = (bidID, error) => ({
  type: 'DOWNLOAD_BID_OFFER_FAILURE',
  payload: { bidID, error },
})

const decryptBidOfferBegin = (bidID) => ({
  type: 'DECRYPT_BID_OFFER',
  payload: { bidID },
})

const decryptBidOfferSuccess = (bidID) => ({
  type: 'DECRYPT_BID_OFFER_SUCCESS',
  payload: { bidID },
})

const decryptBidOfferFailure = (bidID, error) => ({
  type: 'DECRYPT_BID_OFFER_FAILURE',
  payload: { bidID, error },
})

const getBidOfferBegin = (bidID, itemID) => ({
  type: 'GET_BID_OFFER',
  payload: { bidID, itemID },
})

const getBidOfferSuccess = (bidID, itemID, data) => ({
  type: 'GET_BID_OFFER_SUCCESS',
  payload: { bidID, itemID, data }
})

const getBidOfferFailure = (bidID, itemID, error) => ({
  type: 'GET_BID_OFFER_FAILURE',
  payload: { bidID, itemID, error }
})
/****************************/

/****************************/
/* Decypt ask by using AES */
/****************************/
const decryptBidOfferCacheBegin = (itemID, bidID) => ({
  type: 'DECRYPT_BID_OFFER_CACHE',
  payload: { itemID, bidID },
})

const decryptBidOfferCacheSuccess = (itemID, bidID, data) => ({
  type: 'GET_BID_OFFER_CACHE_SUCCESS',
  payload: { itemID, bidID, data }
})

const decryptBidOfferCacheFailure = (itemID, bidID, error) => ({
  type: 'GET_BID_OFFER_CACHE_FAILURE',
  payload: { itemID, bidID, error }
})
/****************************/

export const updateBidDecryptionWaitingList = (data) => ({
  type: 'UPDATE_BID_DECRYPTION_WAITING_LIST',
  payload: data,
})

const downloadBidOffer = (bidDecryptionWaitingList) => (dispatch) => {
  let bidID = null;
  let itemID = null;

  if (bidDecryptionWaitingList[0].bidOffer.length !== 0) {
    let bidOffers = orderBy(bidDecryptionWaitingList[0].bidOffer, 'timestamp', 'desc');
    bidID = bidOffers[0].bidID;
    itemID = bidDecryptionWaitingList[0].itemID;
  } else {
    bidID = 'None';
    itemID = bidDecryptionWaitingList[0].itemID;
  }

  // Begin the task
  dispatch(getBidOfferBegin(bidID, itemID));
  dispatch(downloadBidOfferBegin(bidID));

  return fetch(SELLER_OPTIMISM_API_URL + "download.bid.ciphertext", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      itemID, bidID
    }),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      dispatch(downloadBidOfferFailure(bidID, res.status));
      dispatch(getBidOfferFailure(bidID, itemID, res.status));
      return ""
    }
  }).then(data => {
    if (data !== "") {
      if (data.status === 201) {
        dispatch(downloadBidOfferSuccess(bidID, data.ciphertext));
        return { ciphertext: data.ciphertext, address: data.address, bidID }
      } else {
        return ""
      }
    } else {
      return ""
    }
  })
}

const decryptBidOffer = (itemID, bidID, ciphertext, FHEseed, AESKey, address) => (dispatch) => {

  dispatch(decryptBidOfferBegin(bidID));

  // Web woker
  const workerInstance = cryptoWorker();

  workerInstance.decryptBid(ciphertext, FHEseed, bidID);
  
  workerInstance.addEventListener('message', (message) => {
    if (message.data.status === "success" && 
        message.data.type === "decryptBid" &&
        message.data.bidID === bidID
    ) {
      dispatch(decryptBidOfferSuccess(bidID, message.data.bidCleartext));
      const bidCleartext = message.data.bidCleartext;
      // the last item is 0
      if (bidCleartext[511] === 0) {
        const buyerItemToReceiveAmount = accDiv(bidCleartext[8], Math.pow(10, 5));
        const buyerExchangeRate = accDiv(bidCleartext[9], Math.pow(10, 5));
        dispatch(getBidOfferSuccess(
          bidID, itemID, 
          // data
          { address, buyerItemToReceiveAmount, buyerExchangeRate }
        ));

        /********************************/
        /******* Store the scores *******/
        /********************************/
        const cleartext = { address, buyerItemToReceiveAmount, buyerExchangeRate };

        AESEncrypt(JSON.stringify(cleartext), AESKey).then(ciphertext => {
          
          const bufferCiphertext = ciphertext.ciphertext;
          const bufferIV = ciphertext.iv;
          const base64Ciphertext = {
            ciphertext: encode(bufferCiphertext),
            iv: encode(bufferIV),
          }

          // store ciphertext
          let bidOfferDataCache = localStorage.getItem("bidOfferData");
          if (bidOfferDataCache) bidOfferDataCache = JSON.parse(bidOfferDataCache);
          if (!bidOfferDataCache) {
            bidOfferDataCache = {[itemID]: {[bidID]: base64Ciphertext}}
          } else if (typeof bidOfferDataCache[itemID] === 'undefined') {
            bidOfferDataCache = {
              ...bidOfferDataCache,
              [itemID]: {[bidID]: base64Ciphertext},
            }
          } else {
            bidOfferDataCache = {
              ...bidOfferDataCache,
              [itemID]: {
                ...bidOfferDataCache[itemID],
                [bidID]: base64Ciphertext,
              },
            }
          }
  
          localStorage.setItem("bidOfferData", JSON.stringify(bidOfferDataCache));
        })


        oracleService.publish({
          UUID: bidID,
          amountA: buyerItemToReceiveAmount,
          amountB: accMul(buyerExchangeRate, buyerItemToReceiveAmount).toFixed(5),
          symbolA: arrayToCoin(bidCleartext.slice(0, 4)),
          symbolB: arrayToCoin(bidCleartext.slice(4, 8)),
          source: 'OMG Varna',
        });

      } else {
        dispatch(getBidOfferFailure(bidID, itemID, 400));
      }
    } else if (message.data.bidID === bidID) {
      dispatch(decryptBidOfferFailure(bidID, 404));
      dispatch(getBidOfferFailure(bidID, itemID, 400));
    }
  });
}

export const getBidOffer = (bidDecryptionWaitingList, FHEseed, AESKey) => (dispatch) => {
  dispatch(downloadBidOffer(bidDecryptionWaitingList)).then(data => {
    if (data !== "") {
      dispatch(decryptBidOffer(
        bidDecryptionWaitingList[0].itemID, 
        data.bidID, 
        data.ciphertext, 
        FHEseed, 
        AESKey, 
        data.address,
      ));
    }
  })
}

export const findBidDecryptWaitingList = (itemOpenOrClosed, itemOpenList) => {
  let bidOfferDataCache = localStorage.getItem("bidOfferData");
  if (bidOfferDataCache) bidOfferDataCache = JSON.parse(bidOfferDataCache);
  // input bid offer
  let bidDecryptionWaitingList = [];
  // only upload the bidID accepted by seller
  for (let eachAskOpen of itemOpenList) {
    if (itemOpenOrClosed[eachAskOpen].bidOffer.length !== 0){
      let bidOfferData = [];
      if (bidOfferDataCache) {
        // have cache
        for (let eachBidOffer of itemOpenOrClosed[eachAskOpen].bidOffer) {
          // check if the data is already in cache
          if (bidOfferDataCache[eachAskOpen]) {
            if (!bidOfferDataCache[eachAskOpen][eachBidOffer.bidID]) {
              bidOfferData.push(eachBidOffer);
            }
          } else {
            bidOfferData = itemOpenOrClosed[eachAskOpen].bidOffer
          }
        }
        if (bidOfferData.length !== 0) {
          bidDecryptionWaitingList.push({itemID: eachAskOpen, bidOffer: bidOfferData});
        }
      } else {
        bidDecryptionWaitingList.push({itemID: eachAskOpen, bidOffer: itemOpenOrClosed[eachAskOpen].bidOffer});
      }
    }
  }
  return bidDecryptionWaitingList
}

export const decryptBidOfferCache = (itemOpenOrClosed, itemOpenList, bidOfferDataCache, decryptBidOfferCacheError, AESKey) => async (dispatch) => {

  let BidOfferInCache = false;
  let BidOfferDecrypted = true;
  for (let eachAskOpen of itemOpenList) {
    if (itemOpenOrClosed[eachAskOpen].bidOffer.length !== 0){
      for (let eachBidOffer of itemOpenOrClosed[eachAskOpen].bidOffer) {
        // check whether cache has data
        if (bidOfferDataCache[eachAskOpen]) {
          if (bidOfferDataCache[eachAskOpen][eachBidOffer.bidID]) {
            BidOfferInCache = true;
          }
        }
        // check whether data is decrpyted
        if (decryptBidOfferCacheError[eachAskOpen]) {
          if (typeof decryptBidOfferCacheError[eachAskOpen][eachBidOffer.bidID] === 'undefined') {
            BidOfferDecrypted = false;
          }
        } else {
          BidOfferDecrypted = false;
        }

        if (BidOfferInCache && !BidOfferDecrypted) {
            dispatch(decryptBidOfferCacheBegin(eachAskOpen, eachBidOffer.bidID));

            try{
              const base64Ciphertext = bidOfferDataCache[eachAskOpen][eachBidOffer.bidID].ciphertext;
              const base64IV = bidOfferDataCache[eachAskOpen][eachBidOffer.bidID].iv;
              const ciphertText = {
                ciphertext: decode(base64Ciphertext),
                iv: decode(base64IV),
              }

              const cleartextString = await AESDecrypt(ciphertText, AESKey);
              const cleartext = JSON.parse(cleartextString);
              if (typeof cleartext === 'object' && cleartext !== null) {
                dispatch(decryptBidOfferCacheSuccess(eachAskOpen, eachBidOffer.bidID, cleartext));
              } else {
                dispatch(decryptBidOfferCacheFailure(eachAskOpen, eachBidOffer.bidID, 'Error'));
              }
            } catch (error) {
              dispatch(decryptBidOfferCacheFailure(eachAskOpen, eachBidOffer.bidID, 'Error'));
            }
          }
      }
    }
  }

}