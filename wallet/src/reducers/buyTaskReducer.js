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

  taskStatus: 'stop', // run || stop
  taskStatusLoad: {},
  taskStatusError: {},
  taskStatusBidID: null, // the current task's bidID

  downloadNextItemLoad: {},
  downloadNextItemError: {},

  bidEncrypted: {},
  generateOfferLoad: {},
  generateOfferError: {},

  uploadBidLoad: {},
  uploadBidError: {},
};

function buyTaskReducer (state = initialState, action) {

  switch(action.type) {
    case 'START_BUY_TASK':
      return {
        ...state,
        taskStatus: 'run', // run || stop
        taskStatusBidID: action.payload.bidID,
        taskStatusLoad: {[action.payload.bidID]: true},
        taskStatusError: {[action.payload.bidID]: null},
      }
    case 'START_BUY_TASK_SUCCESS':
      return {
        ...state,
        taskStatus: 'stop', // run || stop
        taskStatusBidID: action.payload.bidID,
        taskStatusLoad: {[action.payload.bidID]: false},
        taskStatusError: {[action.payload.bidID]: false},
      }
    case 'START_BUY_TASK_FAILURE':
      return {
        ...state,
        taskStatus: 'stop', // run || stop
        taskStatusLoad: {[action.payload.bidID]: false},
        taskStatusError: {[action.payload.bidID]: action.payload.error},
      }
    case 'DOWNLOAD_NEXT_ITEM':
      return {
        ...state,
        downloadNextItemLoad: {[action.payload.bidID]: true},
        downloadNextItemError: {[action.payload.bidID]: null},
      }
    case 'DOWNLOAD_NEXT_ITEM_SUCCESS':
      return {
        ...state,
        downloadNextItemLoad: {[action.payload.bidID]: false},
        downloadNextItemError: {[action.payload.bidID]: false},
      }
    case 'DOWNLOAD_NEXT_ITEM_FAILURE':
      return {
        ...state,
        downloadNextItemLoad: {[action.payload.bidID]: false},
        downloadNextItemError: {[action.payload.bidID]: action.payload.error},
      }
    case 'GENERATE_OFFER':
      return {
        ...state,
        bidEncrypted: {},
        generateOfferLoad: {[action.payload.bidID]: true},
        generateOfferError: {[action.payload.bidID]: null},
      }
    case 'GENERATE_OFFER_SUCCESS':
      return {
        ...state,
        bidEncrypted: {[action.payload.bidID]: action.payload.ciphertext},
        generateOfferLoad: {[action.payload.bidID]: false},
        generateOfferError: {[action.payload.bidID]: false},
      }
    case 'GENERATE_OFFER_FAILURE':
      return {
        ...state,
        bidEncrypted: {},
        generateOfferLoad: {[action.payload.bidID]: false},
        generateOfferError: {[action.payload.bidID]: action.payload.error},
      }
    case 'UPLOAD_BID':
      return {
        ...state,
        uploadBidLoad: {[action.payload.bidID]: true},
        uploadBidError: {[action.payload.bidID]: null},
      }
    case 'UPLOAD_BID_SUCCESS':
      return {
        ...state,
        uploadBidLoad: {[action.payload.bidID]: false},
        uploadBidError: {[action.payload.bidID]: false},
      }
    case 'UPLOAD_BID_FAILURE':
      return {
        ...state,
        uploadBidLoad: {[action.payload.bidID]: false},
        uploadBidError: {[action.payload.bidID]: action.payload.error},
      }
    default:
      return state;
  }
}

export default buyTaskReducer;