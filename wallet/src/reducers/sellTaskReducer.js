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

let bidOfferDataCache = localStorage.getItem("bidOfferData");
if (bidOfferDataCache) bidOfferDataCache = JSON.parse(bidOfferDataCache);

const initialState = {
  bidDecryptionWaitingList: [],
  currentDecryptBiditemID: null,
  currentDecryptBidBidID: null,

  bidOfferFileDownload: {},
  downloadBidOfferLoad: {},
  downloadBidOfferError: {},

  decryptBidOfferLoad: {},
  decryptBidOfferError: {},

  decryptBidOfferCacheLoad: {},
  decryptBidOfferCacheError: {},

  bidOfferData: {},
  bidOfferDataCache: bidOfferDataCache ? bidOfferDataCache : {},
  bidOfferDataLoad: {},
  bidOfferDataError: {},
};

function sellTaskReducer (state = initialState, action) {
  switch (action.type) {
    case 'DOWNLOAD_BID_OFFER':
      return {
        ...state,
        bidOfferFileDownload: {},
        downloadBidOfferLoad: {[action.payload.bidID]: true},
        downloadBidOfferError: {[action.payload.bidID]: false},
      }
    case 'DOWNLOAD_BID_OFFER_SUCCESS':
      return {
        ...state,
        bidOfferFileDownload: {[action.payload.bidID]: action.payload.file},
        downloadBidOfferLoad: {[action.payload.bidID]: false},
        downloadBidOfferError: {[action.payload.bidID]: false},
      }
    case 'DOWNLOAD_BID_OFFER_FAILURE':
      return {
        ...state,
        downloadBidOfferLoad: {[action.payload.bidID]: false},
        downloadBidOfferError: {[action.payload.bidID]: action.payload.error},
      }
    case 'DECRYPT_BID_OFFER':
      return {
        ...state,
        decryptBidOfferLoad: {[action.payload.bidID]: true},
        decryptBidOfferError: {[action.payload.bidID]: false},
      }
    case 'DECRYPT_BID_OFFER_SUCCESS':
      return {
        ...state,
        decryptBidOfferLoad: {[action.payload.bidID]: false},
        decryptBidOfferError: {[action.payload.bidID]: false},
      }
    case 'DECRYPT_BID_OFFER_FAILURE':
      return {
        ...state,
        decryptBidOfferLoad: {[action.payload.bidID]: false},
        decryptBidOfferError: {[action.payload.bidID]: action.payload.error},
      }    
    case 'GET_BID_OFFER':
      var bidOfferData = JSON.parse(JSON.stringify(state.bidOfferData));
      var bidOfferDataLoad = JSON.parse(JSON.stringify(state.bidOfferDataLoad));
      var bidOfferDataError = JSON.parse(JSON.stringify(state.bidOfferDataError));
      var bidID = action.payload.bidID;
      var itemID = action.payload.itemID; 

      if (typeof bidOfferData[itemID] === 'undefined') {
        bidOfferData = {
          ...bidOfferData,
          [itemID]: {[bidID]: null},
        }
        bidOfferDataLoad = {
          ...bidOfferDataLoad,
          [itemID]: {[bidID]: true},
        }
        bidOfferDataError = {
          ...bidOfferDataError,
          [itemID]: {[bidID]: null}
        }
      } else {
        bidOfferData = {
          ...bidOfferData,
          [itemID]: {
            ...bidOfferData[itemID],
            [bidID]: null,
          },
        }
        bidOfferDataLoad = {
          ...bidOfferDataLoad,
          [itemID]: {
            ...bidOfferDataLoad[itemID],
            [bidID]: true,
          },
        }
        bidOfferDataError = {
          ...bidOfferDataError,
          [itemID]: {
            ...bidOfferDataError[itemID],
            [bidID]: null,
          }
        }
      }

      return {
        ...state,
        currentDecryptBiditemID: itemID,
        currentDecryptBidBidID: bidID,
        bidOfferData,
        bidOfferDataLoad,
        bidOfferDataError,
      }
    case 'GET_BID_OFFER_SUCCESS': 

      bidOfferData = JSON.parse(JSON.stringify(state.bidOfferData));
      bidOfferDataLoad = JSON.parse(JSON.stringify(state.bidOfferDataLoad));
      bidOfferDataError = JSON.parse(JSON.stringify(state.bidOfferDataError));
      bidID = action.payload.bidID;
      itemID = action.payload.itemID; 
      
      bidOfferData[itemID][bidID] = action.payload.data;
      bidOfferDataLoad[itemID][bidID] = false;
      bidOfferDataError[itemID][bidID] = false;

      return {
        ...state,
        bidOfferData,
        bidOfferDataLoad,
        bidOfferDataError,
      }
    case 'GET_BID_OFFER_FAILURE':

      bidOfferData = JSON.parse(JSON.stringify(state.bidOfferData));
      bidOfferDataLoad = JSON.parse(JSON.stringify(state.bidOfferDataLoad));
      bidOfferDataError = JSON.parse(JSON.stringify(state.bidOfferDataError));
      bidID = action.payload.bidID;
      itemID = action.payload.itemID; 
      
      bidOfferDataLoad[itemID][bidID] = false;
      bidOfferDataError[itemID][bidID] = action.payload.error;

      return {
        ...state,
        bidOfferData,
        bidOfferDataLoad,
        bidOfferDataError,
      }

    /*********************************/
    /********* Download Data *********/
    /********* AES Decryption ********/
    /*********************************/
    case 'DECRYPT_BID_OFFER_CACHE':
      bidOfferData = JSON.parse(JSON.stringify(state.bidOfferData));
      var decryptBidOfferCacheLoad = JSON.parse(JSON.stringify(state.decryptBidOfferCacheLoad));
      var decryptBidOfferCacheError = JSON.parse(JSON.stringify(state.decryptBidOfferCacheError));
      bidID = action.payload.bidID;
      itemID = action.payload.itemID; 

      if (typeof bidOfferData[itemID] === 'undefined') {
        bidOfferData = {
          ...bidOfferData,
          [itemID]: {[bidID]: null},
        }
        decryptBidOfferCacheLoad = {
          ...decryptBidOfferCacheLoad,
          [itemID]: {[bidID]: true},
        }
        decryptBidOfferCacheError = {
          ...decryptBidOfferCacheError,
          [itemID]: {[bidID]: null}
        }
      } else {
        bidOfferData = {
          ...bidOfferData,
          [itemID]: {
            ...bidOfferData[itemID],
            [bidID]: null,
          },
        }
        decryptBidOfferCacheLoad = {
          ...decryptBidOfferCacheLoad,
          [itemID]: {
            ...decryptBidOfferCacheLoad[itemID],
            [bidID]: true,
          },
        }
        decryptBidOfferCacheError = {
          ...decryptBidOfferCacheError,
          [itemID]: {
            ...decryptBidOfferCacheError[itemID],
            [bidID]: null,
          }
        }
      }

      return {
        ...state,
        bidOfferData,
        decryptBidOfferCacheLoad,
        decryptBidOfferCacheError,
      }
    case 'GET_BID_OFFER_CACHE_SUCCESS': 

      bidOfferData = JSON.parse(JSON.stringify(state.bidOfferData));
      decryptBidOfferCacheLoad = JSON.parse(JSON.stringify(state.decryptBidOfferCacheLoad));
      decryptBidOfferCacheError = JSON.parse(JSON.stringify(state.decryptBidOfferCacheError));
      bidID = action.payload.bidID;
      itemID = action.payload.itemID; 

      bidOfferData[itemID][bidID] = action.payload.data;
      decryptBidOfferCacheLoad[itemID][bidID] = false;
      decryptBidOfferCacheError[itemID][bidID] = false;

      return {
        ...state,
        bidOfferData,
        decryptBidOfferCacheLoad,
        decryptBidOfferCacheError,
      }
    case 'GET_BID_OFFER_CACHE_FAILURE':

      decryptBidOfferCacheLoad = JSON.parse(JSON.stringify(state.decryptBidOfferCacheLoad));
      decryptBidOfferCacheError = JSON.parse(JSON.stringify(state.decryptBidOfferCacheError));
      bidID = action.payload.bidID;
      itemID = action.payload.itemID; 
      
      decryptBidOfferCacheLoad[itemID][bidID] = false;
      decryptBidOfferCacheError[itemID][bidID] = action.payload.error;

      return {
        ...state,
        decryptBidOfferCacheLoad,
        decryptBidOfferCacheError,
      }
    
    case 'UPDATE_BID_DECRYPTION_WAITING_LIST':
      return {
        ...state,
        bidDecryptionWaitingList: action.payload,
      }
    default:
      return state;
  }
}

export default sellTaskReducer;