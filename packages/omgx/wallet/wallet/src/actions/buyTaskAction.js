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
import networkService from 'services/networkService';

import { openError } from './uiAction';

import { BUYER_OPTIMISM_API_URL } from '../Settings';

const startBuyTaskBegin = (bidID) => ({
  type: 'START_BUY_TASK',
  payload: { bidID },
})

const startBuyTaskSuccess = (bidID) => ({
  type: 'START_BUY_TASK_SUCCESS',
  payload: { bidID },
})

const startBuyTaskFailure = (bidID, error) => ({
  type: 'START_BUY_TASK_FAILURE',
  payload: { bidID, error },
})

const downloadNextItemBegin = (bidID) => ({
  type: 'DOWNLOAD_NEXT_ITEM',
  payload: { bidID }
})

const downloadNextItemSuccess = (bidID, files) => ({
  type: 'DOWNLOAD_NEXT_ITEM_SUCCESS',
  payload: { bidID, files },
})

const downloadNextItemFailure = (bidID, error) => ({
  type: 'DOWNLOAD_NEXT_ITEM_FAILURE',
  payload: { bidID, error },
})

const generateOfferBegin = (bidID) => ({
  type: 'GENERATE_OFFER',
  payload: { bidID },
})

const generateOfferSuccess = (bidID, ciphertext) => ({
  type: 'GENERATE_OFFER_SUCCESS',
  payload: { bidID, ciphertext }
})

const generateOfferFailure = (bidID, error) => ({
  type: 'GENERATE_OFFER_FAILURE',
  payload: { bidID, error }
})

const uploadBidBegin = (bidID) => ({
  type: 'UPLOAD_BID',
  payload: { bidID },
})

const uploadBidSuccess = (bidID) => ({
  type: 'UPLOAD_BID_SUCCESS',
  payload: { bidID },
})

const uploadBidFailure = (bidID, error) => ({
  type: 'UPLOAD_BID_FAILURE',
  payload: { bidID, error },
})

export const downloadNextItem = (bidID) => (dispatch) => {

  dispatch(downloadNextItemBegin(bidID));

  return fetch(BUYER_OPTIMISM_API_URL + "download.item.ciphertext", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      bidID
    }),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      //getting lots of 400 errors here
      dispatch(downloadNextItemFailure(bidID, res.status));
      return ""
    }
  }).then(data => {
    if (data !== "") {
      if (data.status !== 201) {
        dispatch(downloadNextItemFailure(bidID, data.status));
      } else {
        dispatch(downloadNextItemSuccess(bidID, data));
      }
    }
    return { status: data.status, data }
  })
}

const uploadBid = (bidID, itemID, ciphertext) => (dispatch) => {

  dispatch(uploadBidBegin(bidID));
  // Generate hashed address
  const address = networkService.account;

  return fetch(BUYER_OPTIMISM_API_URL + "upload.bid.seller", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      bidID, itemID, ciphertext, address,
    }),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      dispatch(uploadBidFailure(bidID, res.status));
      dispatch(startBuyTaskFailure(bidID, 500));
      return ""
    }
  }).then(data => {
    if (data !== "") {

      //this controls e.g. run | stop
      /*
        taskStatus: 'stop', // run || stop
        taskStatusLoad: {[action.payload.bidID]: false},
        taskStatusError: {[action.payload.bidID]: false},
      */
      dispatch(startBuyTaskSuccess(bidID));

      //this controls
      /*
          case 'UPLOAD_BID_SUCCESS':
      return {
        ...state,
        uploadBidLoad: {[action.payload.bidID]: false},
        uploadBidError: {[action.payload.bidID]: false},
      */
      dispatch(uploadBidSuccess(bidID));
    }
    return { status: data.status }
  })
}

export const startBuyTask = (bid, bidID) => (dispatch) => {

  //console.log(`running task bidID: ${bidID}`);

  dispatch(startBuyTaskBegin(bidID));

  dispatch(downloadNextItem(bidID)).then(res => {
    if (res.status !== 201) {
      dispatch(startBuyTaskFailure(bidID, res.status));
    } else {
      dispatch(generateOffer(bid, bidID, res.data.itemID, res.data.publicKey));
    }
  })
}

const generateOffer = (bid, bidID, itemID, publicKey) => (dispatch) => {
  
  dispatch(generateOfferBegin(bidID));

  // Web worker
  const workerInstance = cryptoWorker();

  workerInstance.generateOffer(bid, bidID, publicKey);

  workerInstance.addEventListener('message', (message) => {
    if (message.data.status === "success" && 
        message.data.type === "generateOffer" && 
        message.data.bidID === bidID
    ) {
      dispatch(generateOfferSuccess(bidID, message.data.bidCiphertext));
      dispatch(uploadBid(bidID, itemID, message.data.bidCiphertext));
    } else if (message.data.status === "failure") {
      dispatch(generateOfferFailure(bidID, 404));
      dispatch(openError("Failed to encrypt your offer"));
      dispatch(startBuyTaskFailure(bidID, 500));
    }
  });
}
