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

const initialState = {

  masterBidList: {},
  encryptedBidCache: {},
  getBidCTForBuyer: {},

  numberOfItemsOnVarna: 0,
  numberOfItemsOnVarnaLoad: false,
  numberOfItemsOnVarnaError: null,

  encryptBidForBuyerLoad: false,
  encryptBidForBuyerError: null,

  uploadBuyerBidAndStatusLoad: false,
  uploadBuyerBidAndStatusError: null,

  configureBidToPlasmaLoad: false,
  configureBidToPlasmaError: null,

  bidOpenOrClosed: {},
  bidOpenOrClosedLoad: false,
  bidOpenOrClosedError: null,

  bidMakingTasks: {},
  bidMakingTasksLoad: false,
  bidMakingTasksError: null,

  bidCiphertextForBuyer: {},
  downloadBidCiphertextForBuyerLoad: {},
  downloadBidCiphertextForBuyerError: {},

  decryptedBid: {},
  decryptedBidLoad: {},
  decryptedBidError: {},

  bidAcceptData: {},
  getBidAcceptDataLoad: false,
  getBidAcceptDataError: null,

  buyerApproveSwap: [],
  buyerApproveSwapLoad: {},
  buyerApproveSwapError: {},
};

function buyReducer (state = initialState, action) {

  switch (action.type) {
    case 'GET_BUYER_ITEM_NUMBER':
      return {
        ...state,
        numberOfItemsOnVarnaLoad: true,
        numberOfItemsOnVarnaError: null,
      }
    case 'GET_BUYER_ITEM_NUMBER_SUCCESS':
      return {
        ...state,
        numberOfItemsOnVarna: action.payload.askActiveNumber,
        numberOfItemsOnVarnaLoad: false,
        numberOfItemsOnVarnaError: false,
      }
    case 'GET_BUYER_ITEM_NUMBER_FAILURE':
      return {
        ...state,
        numberOfItemsOnVarnaLoad: false,
        numberOfItemsOnVarnaError: action.payload,
      }
    case 'ENCRYPT_BID_FOR_BUYER':
      return {
        ...state,
        encryptBidForBuyerLoad: true,
        encryptBidForBuyerError: null,
      }
    case 'ENCRYPT_BID_FOR_BUYER_SUCCESS':
      return {
        ...state,
        encryptBidForBuyerLoad: false,
        encryptBidForBuyerError: false,
      }
    case 'ENCRYPT_BID_FOR_BUYER_FAILURE':
      return {
        ...state,
        encryptBidForBuyerLoad: false,
        encryptBidForBuyerError: action.payload,
      }
    case 'UPLOAD_BUYER_BID_AND_STATUS':
      return {
        ...state,
        uploadBuyerBidAndStatusLoad: true,
        uploadBuyerBidAndStatusError: null,
      }
    case 'UPLOAD_BUYER_BID_AND_STATUS_SUCCESS':
      return {
        ...state,
        uploadBuyerBidAndStatusLoad: false,
        uploadBuyerBidAndStatusError: false,
      }
    case 'UPLOAD_BUYER_BID_AND_STATUS_FAILURE':
      return {
        ...state,
        uploadBuyerBidAndStatusLoad: false,
        uploadBuyerBidAndStatusError: action.payload,
      }
    case 'CONFIGURE_BID_TO_PLASMA':
      return {
        ...state,
        configureBidToPlasmaLoad: true,
        configureBidToPlasmaError: null,
      }
    case 'CONFIGURE_BID_TO_PLASMA_SUCCESS':
      return {
        ...state,
        configureBidToPlasmaLoad: false,
        configureBidToPlasmaError: false,
      }
    case 'CONFIGURE_BID_TO_PLASMA_FAILURE':
      return {
        ...state,
        configureBidToPlasmaLoad: false,
        configureBidToPlasmaError: action.payload,
      }
    case 'BID_OPEN_OR_CLOSED':
      return {
        ...state,
        bidOpenOrClosedLoad: true,
        bidOpenOrClosedError: null,
      }
    case 'BID_OPEN_OR_CLOSED_SUCCESS':

      let bids = action.payload;
      let masterBidList = {}
      let masterBidListState = {...state.masterBidList}

      Object.keys(bids).forEach((bidID, index) => {
        if (bids[bidID].status === 'active') {
          if (masterBidListState[bidID]) {
            masterBidList[bidID] = masterBidListState[bidID]
          } else {
            masterBidList[bidID] = {
              status: 'active',
              downloaded: false,
              decrypted: false,
              loading: false,
              createdAt: bids[bidID].createdAt,
              updatedAt: bids[bidID].updatedAt,
            }
          }
        }
      })

      return {
        ...state,
        masterBidList,
        bidOpenOrClosedLoad: false,
        bidOpenOrClosedError: false,
      }
    case 'BID_OPEN_OR_CLOSED_FAILURE':
      return {
        ...state,
        bidOpenOrClosed: {},
        bidOpenOrClosedLoad: false,
        bidOpenOrClosedError: action.payload,
      }
    case 'GET_BID_MAKING_TASK_LIST_BEGIN':
      return {
        ...state,
        bidMakingTasksLoad: true,
        bidMakingTasksError: null,
      }
    case 'GET_BID_MAKING_TASK_LIST_SUCCESS':
      return {
        ...state,
        bidMakingTasks: action.payload,
        bidMakingTasksLoad: false,
        bidMakingTasksError: false,
      }
    case 'GET_BID_MAKING_TASK_LIST_FAILURE':
      return {
        ...state,
        bidMakingTasksLoad: false,
        bidMakingTasksError: action.payload,
      }
    case 'CLOSE_BID_SUCCESS':
      console.log("close_success")
      return {
        ...state,
        masterBidList: {
          ...state.masterBidList,
          [action.payload.bidID]: {
            status: 'closed',
            error: false,
          }
        }
      }
    case 'CLOSE_BID_FAILURE':
    console.log("close_failure")
      return {
        ...state
      }
    case 'GET_BID_FOR_BUYER':
      return {
        ...state,
        masterBidList: {
          ...state.masterBidList,
          [action.payload.bidID]: {
            ...state.masterBidList[action.payload.bidID],
            bidInfoClear: false,
            loading: true,
            downloaded: false,
            decrypted: false,
            downloadError: false,
          }
        }
      }

    case 'GET_BID_FOR_BUYER_SUCCESS':
      let decryptionDone = false;
      if(action.payload.plaintext.length > 0) {
        decryptionDone = true;
      } else {
        console.log("Still waiting for decrypt")
      }
      return {
        ...state,
        masterBidList: {
          ...state.masterBidList,
          [action.payload.bidID]: {
            ...state.masterBidList[action.payload.bidID],
            bidInfoClear: decryptionDone ? action.payload.plaintext.slice(0, 12) : [],
            loading: false,
            downloaded: true,
            decrypted: decryptionDone,
            downloadError: false,
          }
        }
      }
    case 'GET_BID_FOR_BUYER_FAILURE':
      return {
        ...state,
        masterBidList: {
          ...state.masterBidList,
          [action.payload.bidID]: {
            ...state.masterBidList[action.payload.bidID],
            bidInfoClear: false,
            loading: false,
            downloaded: true,
            decrypted: false,
            downloadError: action.payload.error,
          }
        }
      }
    case 'BUYER_APPROVE_SWAP':
      return {
        ...state,
        buyerApproveSwapLoad: {
          ...state.buyerApproveSwapLoad,
          [action.payload.bidID]: true
        },
        buyerApproveSwapError: {
          ...state.buyerApproveSwapError,
          [action.payload.bidID]: null
        },
      }
    case 'BUYER_APPROVE_SWAP_SUCCESS':
      return {
        ...state,
        buyerApproveSwap: [
          ...state.buyerApproveSwap, action.payload.bidID
        ],
        buyerApproveSwapLoad: {
          ...state.buyerApproveSwapLoad,
          [action.payload.bidID]: false
        },
        buyerApproveSwapError: {
          ...state.buyerApproveSwapError,
          [action.payload.bidID]: false
        },
      }
    case 'BUYER_APPROVE_SWAP_FAILURE':
      return {
        ...state,
        buyerApproveSwapLoad: {
          ...state.buyerApproveSwapLoad,
          [action.payload.bidID]: false
        },
        buyerApproveSwapError: {
          ...state.buyerApproveSwapError,
          [action.payload.bidID]: action.payload.error
        },
      }
    case 'GET_BID_ACCEPT_DATA':
      return {
        ...state,
        getBidAcceptDataLoad: true,
        getBidAcceptDataError: null,
      }
    case 'GET_BID_ACCEPT_DATA_SUCCESS':
      return {
        ...state,
        bidAcceptData: action.payload,
        getBidAcceptDataLoad: false,
        getBidAcceptDataError: false,
      }
    case 'GET_BID_ACCEPT_DATA_FAILURE':
      return {
        ...state,
        getBidAcceptDataLoad: false,
        getBidAcceptDataError: action.payload,
      }
    default:
      return state;
  }
}

export default buyReducer;