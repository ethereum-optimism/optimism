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
  swapMetaData: {},
  swappedBid: [],
}

function swapReducer (state = initialState, action) {
  switch (action.type) {
    case 'UPDATE_SWAP_META_DATA': 
      let metaDataRow = action.payload;
      let metaDataClean = {}, swappedBid = [], itemIDPartial = null, bidIDPartial = null;
      for (let eachMetaData of metaDataRow) {
        [itemIDPartial, bidIDPartial] = eachMetaData.split('-SWAP-');
        if (metaDataClean[itemIDPartial] === undefined) {
          metaDataClean[itemIDPartial] = [bidIDPartial];
          swappedBid.push(bidIDPartial);
        } else {
          metaDataClean[itemIDPartial].push(bidIDPartial);
          swappedBid.push(bidIDPartial);
        }
      }

      return {
        ...state,
        swapMetaData: metaDataClean,
        swappedBid,
      }
    default:
      return state;
  }
}

export default swapReducer;