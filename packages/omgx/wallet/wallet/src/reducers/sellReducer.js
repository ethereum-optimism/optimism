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

let decryptedItemCache = localStorage.getItem("decryptedItem");

if (decryptedItemCache) {
  decryptedItemCache = JSON.parse(decryptedItemCache);
}

const initialState = {
  generateKeyLoad: false,
  generateKeyError: null,

  uploadKeyLoad: false,
  uploadKeyError: null,

  configureItemToOMGXLoad: false,
  configureItemToOMGXError: null,

  itemOpenOrClosed: {},
  itemOpenList: [],
  itemOpenOrClosedLoad: false,
  itemOpenOrClosedError: null,
  itemOpenOrClosedLoadIndicator: null,

  currentDecryptItemID: null,

  downloadItemCiphertext: {},
  downloadItemCiphertextLoad: {},
  downloadItemCiphertextError: {},

  decryptedItem: {},
  decryptedItemLoad: {},
  decryptedItemError: {},
  goodItemDecrypt: null,

  decryptedItemCache: decryptedItemCache ? decryptedItemCache : {},
  decryptedItemCacheLoad: {},
  decryptedItemCacheError: {},

  deleteItemLoad: {},
  deleteItemError: {},

  acceptBidLoad: {},
  acceptBidError: {},

  sellerAcceptBidData: {},
  getSellerAcceptBidDataLoad: false,
  getSellerAcceptBidDataError: null,
};

function sellReducer (state = initialState, action) {
  switch (action.type) {
    case 'ENCRYPT_ITEM':
      return {
        ...state,
        generateKeyLoad: true,
        generateKeyError: null,
      }
    case 'ENCRYPT_ITEM_SUCCESS':
      return {
        ...state,
        generateKeyLoad: false,
        generateKeyError: false,
      }
    case 'ENCRYPT_ITEM_FAILURE':
      return {
        ...state,
        generateKeyLoad: false,
        generateKeyError: action.payload,
      }
    case 'UPLOAD_ITEM_FILES':
      return {
        ...state,
        uploadKeyLoad: true,
        uploadKeyError: null,
      }
    case 'UPLOAD_ITEM_FILES_SUCCESS':
      return {
        ...state,
        uploadKeyLoad: false,
        uploadKeyError: false,
      }    
    case 'UPLOAD_ITEM_FILES_FAILURE':
      return {
        ...state,
        uploadKeyLoad: false,
        uploadKeyError: action.payload,
      }
    case 'CONFIGURE_ITEM_TO_PLASMA':
      return {
        ...state,
        configureItemToOMGXLoad: true,
        configureItemToOMGXError: null,
      }
    case 'CONFIGURE_ITEM_TO_PLASMA_SUCCESS':
      return {
        ...state,
        configureItemToOMGXLoad: false,
        configureItemToOMGXError: false,
      }
    case 'CONFIGURE_ITEM_TO_PLASMA_FAILURE':
      return {
        ...state,
        configureItemToOMGXLoad: false,
        configureItemToOMGXError: action.payload,
      }
    case 'ITEM_OPEN_OR_CLOSED':

      let itemOpenOrClosedLoadIndicator = state.itemOpenOrClosedLoadIndicator;
      if (state.itemOpenOrClosedLoadIndicator === null) {
        itemOpenOrClosedLoadIndicator = true;
      }
      return {
        ...state,
        itemOpenOrClosedLoad: true,
        itemOpenOrClosedLoadIndicator,
        itemOpenOrClosedError: null,
      }
    case 'ITEM_OPEN_OR_CLOSED_SUCCESS':
      let itemOpenOrClosed = action.payload;
      let itemOpenList = Object.keys(itemOpenOrClosed).reduce((cur, acc) => {
        if (itemOpenOrClosed[acc].status === "active") {
          cur.push(acc);
        }
        return cur
      },[])
      return {
        ...state,
        itemOpenOrClosed,
        itemOpenList,
        itemOpenOrClosedLoad: false,
        itemOpenOrClosedLoadIndicator: false,
        itemOpenOrClosedError: false,
      }
    case 'ITEM_OPEN_OR_CLOSED_FAILURE':
      return {
        ...state,
        itemOpenOrClosed: {},
        itemOpenOrClosedLoad: false,
        itemOpenOrClosedLoadIndicator: false,
        itemOpenOrClosedError: action.payload,
      }
    case 'DOWNLOAD_ITEM_CIPHERTEXT':
      return {
        ...state,
        currentDecryptItemID: action.payload.itemID,
        downloadItemCiphertext: {},
        downloadItemCiphertextLoad: {[action.payload.itemID]: true},
        downloadItemCiphertextError: {[action.payload.itemID]: null},
      }
    case 'DOWNLOAD_ITEM_CIPHERTEXT_SUCCESS':
      return {
        ...state,
        downloadItemCiphertext: {[action.payload.itemID]: action.payload.ciphertext},
        downloadItemCiphertextLoad: {[action.payload.itemID]: false},
        downloadItemCiphertextError: {[action.payload.itemID]: false},
      }
    case 'DOWNLOAD_ITEM_CIPHERTEXT_FAILURE':
      return {
        ...state,
        downloadItemCiphertext: {},
        downloadItemCiphertextLoad: {[action.payload.itemID]: false},
        downloadItemCiphertextError: {[action.payload.itemID]: action.payload.error},
      }

    /* Decrypt ask using FHE */
    case 'DECRYPT_ITEM': 
      return {
        ...state,
        decryptedItemLoad: {
          ...state.decryptedItemLoad, 
          [action.payload.itemID]: true
        },
        decryptedItemError: {
          ...state.decryptedItemError, 
          [action.payload.itemID]: null
        },
      }  
    case 'DECRYPT_ITEM_SUCCESS': 
      return {
        ...state,
        goodItemDecrypt: true,
        decryptedItem: {
          ...state.decryptedItem, 
          [action.payload.itemID]: action.payload.cleartext
        },
        decryptedItemLoad: {
          ...state.decryptedItemLoad, 
          [action.payload.itemID]: false
        },
        decryptedItemError: {
          ...state.decryptedItemError, 
          [action.payload.itemID]: false
        },
      }  
    case 'DECRYPT_ITEM_FAILURE': 
      return {
        ...state,
        goodItemDecrypt: false,
        decryptedItemLoad: {
          ...state.decryptedItemLoad, 
          [action.payload.itemID]: false
        },
        decryptedItemError: {
          ...state.decryptedItemError, 
          [action.payload.itemID]: action.payload.error
        },
      }
    
    /* Decrypt ask AES */
    case 'DECRYPT_ITEM_CACHE': 
      return {
        ...state,
        decryptedItemCacheLoad: {
          ...state.decryptedItemCacheLoad, 
          [action.payload.itemID]: true
        },
        decryptedItemCacheError: {
          ...state.decryptedItemCacheError, 
          [action.payload.itemID]: null
        },
      }  
    case 'DECRYPT_ITEM_CACHE_SUCCESS': 
      return {
        ...state,
        decryptedItem: {
          ...state.decryptedItem, 
          [action.payload.itemID]: action.payload.cleartext
        },
        decryptedItemCacheLoad: {
          ...state.decryptedItemCacheLoad, 
          [action.payload.itemID]: false
        },
        decryptedItemCacheError: {
          ...state.decryptedItemCacheError, 
          [action.payload.itemID]: false
        },
      }  
    case 'DECRYPT_ITEM_CACHE_FAILURE': 
      return {
        ...state,
        // decryptedItem: {
        //   ...state.decryptedItem, 
        //   [action.payload.itemID]: action.payload.error,
        // },
        decryptedItemCacheLoad: {
          ...state.decryptedItemCacheLoad, 
          [action.payload.itemID]: false
        },
        decryptedItemCacheError: {
          ...state.decryptedItemCacheError, 
          [action.payload.itemID]: action.payload.error
        },
      }
    
    case 'UPDATE_DECRYPTED_ITEM':
      return {
        ...state,
        decryptedItem: action.payload,
      }
    case 'DELETE_ITEM':
      return {
        ...state,
        deleteItemLoad: {
          ...state.deleteItemLoad, 
          [action.payload.itemID]: true
        },
        deleteItemError: {
          ...state.deleteItemError, 
          [action.payload.itemID]: null,
        },
      }
    case 'DELETE_ITEM_SUCCESS':
      return {
        ...state,
        deleteItemLoad: {
          ...state.deleteItemLoad, 
          [action.payload.itemID]: false
        },
        deleteItemError: {
          ...state.deleteItemError, 
          [action.payload.itemID]: false,
        },
      }
    case 'DELETE_ITEM_FAILURE':
      return {
        ...state,
        deleteItemLoad: {
          ...state.deleteItemLoad, 
          [action.payload.itemID]: false
        },
        deleteItemError: {
          ...state.deleteItemError, 
          [action.payload.itemID]: action.payload.error,
        },
      }
    case 'ACCEPT_BID':
      return {
        ...state,
        acceptBidLoad: {
          ...state.acceptBidLoad,
          [action.payload.itemID]: {[action.payload.bidID]: true}
        },
        acceptBidError: {
          ...state.acceptBidError,
          [action.payload.itemID]: {[action.payload.bidID]: null}
        },
      }
    case 'ACCEPT_BID_SUCCESS':
      return {
        ...state,
        acceptBidLoad: {
          ...state.acceptBidLoad,
          [action.payload.itemID]: {[action.payload.bidID]: false}
        },
        acceptBidError: {
          ...state.acceptBidError,
          [action.payload.itemID]: {[action.payload.bidID]: false}
        },
      }
    case 'ACCEPT_BID_FAILURE':
      return {
        ...state,
        acceptBidLoad: {
          ...state.acceptBidLoad,
          [action.payload.itemID]: {[action.payload.bidID]: false}
        },
        acceptBidError: {
          ...state.acceptBidError,
          [action.payload.itemID]: {[action.payload.bidID]: action.payload.error}
        },
      }
    case 'UPDATE_RESERVED_UTXO':
      return {
        ...state,
        reservedUTXO: action.payload,
      }
    case 'GET_SELLER_ACCEPT_BID_DATA':
      return {
        ...state,
        getSellerAcceptBidDataLoad: true,
        getSellerAcceptBidDataError: null,
      }
    case 'GET_SELLER_ACCEPT_BID_DATA_SUCCESS':
      return {
        ...state,
        sellerAcceptBidData: action.payload,
        getSellerAcceptBidDataLoad: false,
        getSellerAcceptBidDataError: false,
      }
    case 'GET_SELLER_ACCEPT_BID_DATA_FAILURE':
      return {
        ...state,
        getSellerAcceptBidDataLoad: false,
        getSellerAcceptBidDataError: action.payload,
      }
    default:
      return state;
  }
}

export default sellReducer;