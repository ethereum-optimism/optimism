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

import { ethers } from 'ethers';
import networkService from 'services/networkService';
import cryptoWorker from 'workerize-loader!../cryptoWorker/cryptoWorker'; // eslint-disable-line import/no-webpack-loader-syntax
import md5 from 'md5';

import { openAlert, openError } from './uiAction';

import { BUYER_OPTIMISM_API_URL } from '../Settings';

const get_number_of_items_on_varnaBegin = () => ({
  type: 'GET_BUYER_ITEM_NUMBER',
})

const get_number_of_items_on_varnaSuccess = (data) => ({
  type: 'GET_BUYER_ITEM_NUMBER_SUCCESS',
  payload: data,
})

const get_number_of_items_on_varnaFailure = (data) => ({
  type: 'GET_BUYER_ITEM_NUMBER_FAILURE',
  payload: data,
})

const encryptBidForBuyerBegin = () => ({
  type: 'ENCRYPT_BID_FOR_BUYER',
})

const encryptBidForBuyerSuccess = () => ({
  type: 'ENCRYPT_BID_FOR_BUYER_SUCCESS',
})

const encryptBidForBuyerFailure = (data) => ({
  type: 'ENCRYPT_BID_FOR_BUYER_FAILURE',
  payload: data,
})

const uploadBuyerBidAndStatusBegin = () => ({
  type: 'UPLOAD_BUYER_BID_AND_STATUS',
})

const uploadBuyerBidAndStatusSuccess = () => ({
  type: 'UPLOAD_BUYER_BID_AND_STATUS_SUCCESS',
})

const uploadBuyerBidAndStatusFailure = (data) => ({
  type: 'UPLOAD_BUYER_BID_AND_STATUS_FAILURE',
  payload: data,
})

const configureBidToPlasmaBegin = () => ({
  type: "CONFIGURE_BID_TO_PLASMA",
})

const configureBidToPlasmaSuccess = () => ({
  type: "CONFIGURE_BID_TO_PLASMA_SUCCESS",
})

const configureBidToOMGXFailure = (data) => ({
  type: "CONFIGURE_BID_TO_PLASMA_FAILURE",
  payload: data,
})

const isBidOpenOrClosedBegin = () => ({
  type: 'BID_OPEN_OR_CLOSED'
})

const isBidOpenOrClosedSuccess = (data) => ({
  type: 'BID_OPEN_OR_CLOSED_SUCCESS',
  payload: data,
})

const isBidOpenOrClosedFailure = (data) => ({
  type: 'BID_OPEN_OR_CLOSED_FAILURE',
  payload: data,
})

const getBidMakingTasksBegin = () => ({
  type: 'GET_BID_MAKING_TASK_LIST_BEGIN',
})

const getBidMakingTasksSuccess = (data) => ({
  type: 'GET_BID_MAKING_TASK_LIST_SUCCESS',
  payload: data,
})

const getBidMakingTasksFailure = (data) => ({
  type: 'GET_BID_MAKING_TASK_LIST_FAILURE',
  payload: data,
})

const closeBidSuccess = (bidID) => ({
  type: 'CLOSE_BID_SUCCESS',
  payload: { bidID },
})

const closeBidFailure = (bidID, error) => ({
  type: 'CLOSE_BID_FAILURE',
  payload: { bidID, error }
})

const getBidBuyerBegin = (bidID) => ({
  type: 'GET_BID_FOR_BUYER',
  payload: { bidID },
})

const getBidBuyerSuccess = (bidID, plaintext) => ({
  type: 'GET_BID_FOR_BUYER_SUCCESS',
  payload: { bidID, plaintext }
})

const getBidBuyerFailure = (bidID, error) => ({
  type: 'GET_BID_FOR_BUYER_FAILURE',
  payload: { bidID, error }
})

/****************************/
/* Buyer Accepts Bid data */
/****************************/
const buyerApproveSwapBegin = (bidID) => ({
  type: 'BUYER_APPROVE_SWAP',
  payload: { bidID },
});

const buyerApproveSwapSuccess = (bidID) => ({
  type: 'BUYER_APPROVE_SWAP_SUCCESS',
  payload: { bidID },
});

const buyerApproveSwapFailure = (bidID, error) => ({
  type: 'BUYER_APPROVE_SWAP_FAILURE',
  payload: { bidID, error },
});
/****************************/

/****************************/
/* Seller Accepts Bid data */
/****************************/
const getBidAcceptDataBegin = () => ({
  type: 'GET_BID_ACCEPT_DATA',
})

const getBidAcceptDataSuccess = (data) => ({
  type: 'GET_BID_ACCEPT_DATA_SUCCESS',
  payload: data,
})

const getBidAcceptDataFailure = (data) => ({
  type: 'GET_BID_ACCEPT_DATA_FAILURE',
  payload: data,
})
/****************************/

export const get_number_of_items_on_varna = () => (dispatch) => {
  dispatch(get_number_of_items_on_varnaBegin());

  fetch(BUYER_OPTIMISM_API_URL + 'item.count', {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({}),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      dispatch(get_number_of_items_on_varnaFailure(res.status));
      return ""
    }
  }).then(data => {
    if (data !== "") {
      dispatch(get_number_of_items_on_varnaSuccess(data));
    }
  })
}

/* Upload buyer original bid */
const uploadBuyerBidAndStatus = (bidID, ciphertext, itemToReceive, itemToSend, address) => (dispatch) => {

  dispatch(uploadBuyerBidAndStatusBegin());

  const body = JSON.stringify({ 
    bidID, 
    ciphertext, 
    symbolA: itemToReceive.symbol, 
    symbolB: itemToSend.symbol,
    address,
  });

  return fetch(BUYER_OPTIMISM_API_URL + 'upload.bid.buyer', {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body,
  }).then(res => {
    if (res.status === 201) {
      dispatch(uploadBuyerBidAndStatusSuccess());
      return {status: 201}
    } else {
      dispatch(uploadBuyerBidAndStatusFailure(res.status));
      return {status: res.status}
    }
  })
}

export const isBidOpenOrClosed = () => (dispatch) => {
  // Generate hashed address
  const address = md5(networkService.account);

  dispatch(isBidOpenOrClosedBegin());

  fetch(BUYER_OPTIMISM_API_URL + "download.bid.status", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      address
    }),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      dispatch(isBidOpenOrClosedFailure(res.status));
      return ""
    }
  }).then(data => {
    if (data !== "") {
      dispatch(isBidOpenOrClosedSuccess(data.data));
    }
  })

}

export const getBidMakingTasks = (rescanPlasma = false) => (dispatch) => {

  dispatch(getBidMakingTasksBegin());

  // Generate hashed address
  const address = md5(networkService.account);
  let url = BUYER_OPTIMISM_API_URL + "scan.light";

  if(rescanPlasma) 
    url = BUYER_OPTIMISM_API_URL + "scan.full";

  fetch(url, {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      address
    }),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      dispatch(getBidMakingTasksFailure(res.status));
      if(rescanPlasma) 
        dispatch(openAlert("Failed to update bidding task list"))
      return ""
    }
  }).then(data => {
    if (data !== "") {
      if(rescanPlasma){
        console.log("getBidMakingTasksRefreshed:", data)
        dispatch(getBidMakingTasksSuccess(data));
      } else {
        // console.log("getBidMakingTasks:", data.bidOfferStatus);
        dispatch(getBidMakingTasksSuccess(data.bidOfferStatus));
      }
    }
  })

}

export const closeBid = (bidID) => (dispatch) => {
  // Generate hashed address
  const address = md5(networkService.account);

  fetch(BUYER_OPTIMISM_API_URL + "delete.bid", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      bidID, address
    }),
  }).then(res => {
    if (res.status === 201) {
      dispatch(closeBidSuccess(bidID));
    } else {
      dispatch(closeBidFailure(bidID, res.status));
    }
  })
}

export const getBidBuyer = (bidID, password) => (dispatch) => {
  dispatch(getBidBuyerBegin(bidID));

  return fetch(BUYER_OPTIMISM_API_URL + "download.bid.ciphertext", {
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
      return res.json();
    } else {
      dispatch(getBidBuyerFailure(bidID, res.json()));
      return ""
    }
  }).then(data => {
    if (Object.keys(data.ciphertext).length !== 0) {
      dispatch(decryptBidForBuyer(bidID, data.ciphertext, password));
      dispatch(getBidBuyerSuccess(bidID, [])); //plaintext is pending - can't fill in yet
    } else {
      dispatch(getBidBuyerFailure(bidID, 400));
    }
    return data
  })
}

const decryptBidForBuyer = (bidID, ciphertext, password) => (dispatch) => {

  // Web worker
  const workerInstance = cryptoWorker();

  workerInstance.decryptBid(ciphertext, password, bidID);

  workerInstance.addEventListener('message', (message) => {
    if (message.data.status === "success" && 
        message.data.type === "decryptBid" && 
        message.data.bidID === bidID
    ) {
      //console.log("export const getBidBuyer: decrypting is done!")
      dispatch(getBidBuyerSuccess(bidID, message.data.bidCleartext));
    } else if (message.data.status === "failure") {
      //console.log("export const getBidBuyer: decrypting failed!")
      dispatch(getBidBuyerFailure(bidID, 404));
    }
  });
}

export const acceptSellerSwap = (cMD) => async (dispatch) => {

  /* hashID, bidID, sender, hashResult */
  /*
    UUID,
    agreeAmount,
    agreeExchangeRate,
    buyerItemToReceive,
    buyerItemToSend,
    sellerItemToSend: buyerItemToReceive,
    sellerItemToReceive: buyerItemToSend,
    bidID: data.bidID,
    itemID: data.bidAcceptDetails.itemID,
    address: data.bidAcceptDetails.sellerAddress,
    type: 'buyerAccept'
   */

  dispatch(buyerApproveSwapBegin(cMD.bidID));

  try {
    const swapID = ethers.utils.soliditySha3(cMD.UUID);
    const swapStatus = await networkService.AtomicSwapContract.close(
      swapID,
    );
    const swapRes = await swapStatus.wait();

    if (swapRes) {
      closeBidOffer(cMD.UUID);
      console.log({ "swap receipt": swapRes });
      dispatch(buyerApproveSwapSuccess(cMD.bidID));
      dispatch(openAlert("Swap Completed"));
    }
  } catch (error) {
    console.log(error);
    dispatch(buyerApproveSwapFailure(cMD.bidID, 'Swap failed! - check the log for more information'));
  }
}

export const listBid = (
    itemToReceive, 
    itemToReceiveAmount, 
    itemToSend,
    buyerExchangeRate, 
    FHEseed,
  ) => async (dispatch) => {

  console.log("listBid: Starting the bid listing process")
  var cryptoWorkerThreadID = crypto.getRandomValues(new Uint32Array(1)).toString(16);

  dispatch(encryptBidForBuyerBegin());

  const workerInstance = cryptoWorker();

  workerInstance.encryptBid(
    itemToReceive, 
    itemToReceiveAmount, 
    itemToSend, 
    buyerExchangeRate, 
    FHEseed, 
    cryptoWorkerThreadID
  );

  await workerInstance.addEventListener('message', async (message) => {
    if (message.data.status === "success" && 
        message.data.type === "encryptBid" && 
        message.data.cryptoWorkerThreadID === cryptoWorkerThreadID
    ) {

      dispatch(encryptBidForBuyerSuccess());
      dispatch(configureBidToPlasmaBegin());

      // Generate BidID
      const bidID = md5(JSON.stringify(message.data.bidCiphertext));
      // Generate hashed address
      const address = md5(networkService.account);

      /*****************************************/
      /****** Removed the smart contract ******/
      /****************************************/
      // fire the smart contract
      // try {
      //   const tx = await networkService.VarnaPoolContract.listBid(
      //     address,
      //     bidID,
      //     `${itemToSend.symbol}-${itemToReceive.symbol}/${bidID}`,
      //     new Date().getTime(),
      //   );
      //   if (tx === '' || typeof tx === undefined) {
      //     dispatch(configureBidToOMGXFailure(404));
      //     dispatch(openError("Failed to broadcast your bid"));
      //     return
      //   }
      // } catch {
      //   dispatch(configureBidToOMGXFailure(404));
      //   dispatch(openError("Failed to broadcast your bid"));
      //   return 
      // }

      const uploadStatus = await dispatch(uploadBuyerBidAndStatus(
        bidID, 
        message.data.bidCiphertext,
        itemToReceive,
        itemToSend,
        address,
      ));

      if (uploadStatus.status === 201) {
        dispatch(configureBidToPlasmaSuccess());
        dispatch(uploadBuyerBidAndStatusSuccess());
        dispatch(openAlert("New bid listed"));
      } else {
        dispatch(configureBidToOMGXFailure(404));
        dispatch(uploadBuyerBidAndStatusFailure(uploadStatus.status));
        dispatch(openError("Failed to broadcast your bid"));
      }

    } else if (message.data.status === "failure") {
      dispatch(encryptBidForBuyerFailure(404));
      dispatch(openError("Failed to encrypt your bid"));
    }
  })
}

export const getBidAcceptData = (bidIDList) => (dispatch) => {
  dispatch(getBidAcceptDataBegin());

  return fetch(BUYER_OPTIMISM_API_URL + "download.agreement", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      bidIDList, address: networkService.account,
    }),
  }).then(res => {
    if (res.status === 201) {
      return res.json()
    } else {
      dispatch(getBidAcceptDataFailure(res.status));
      return ""
    }
  }).then(data => {
    if (data !== "") {
      dispatch(getBidAcceptDataSuccess(data));
    }
  })
}

const closeBidOffer = (UUID) => {
  fetch(BUYER_OPTIMISM_API_URL + "close.agreement", {
    method: "POST",
    headers: {
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ UUID }),
  }).then(res => {
    return res.status;
  })
}