/*
  Utility functions fo OMG Plasma Users 
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

import { orderBy } from 'lodash';

export const findHashcasts = (transaction) => {

  const orderedTransactions = orderBy(transaction, i => i.block.timestamp, 'desc');

  const hashcasts = [];

  orderedTransactions.forEach(element => {

    //todo - write this elegantly with filters rather than silly if()s
    if(typeof element.metadata === 'undefined') return;
    if(element.metadata === '') return;
    if(element.metadata === 'Merge UTXOs') return;
    if(element.metadata === 'atomic swap') return;
    if(element.metadata.includes('--')) return;
    if(element.metadata.includes('-SWAP-')) return;
    if(element.metadata.includes('<->')) return;
    if(element.metadata.includes('Hola')) return;

    //console.log("ownerS:", element.inputs)
    //console.log("ownerR:", element.outputs)

    //a plain transfer has the sender wallet as as element.inputs[0].owner, and the 
    //recipient wallet address is element.outputs[1].owner
    if(typeof element.inputs !== 'undefined' && typeof element.outputs !== 'undefined') {
      if(element.inputs.length > 0 && element.outputs.length > 1) {
        const sender    = element.inputs[element.inputs.length-1].owner;
        const recipient = element.outputs[element.outputs.length-1].owner;
        if ( sender !== recipient ) //it's a 'real' transfer among two distinct wallets
          //so we don't care about it
          return;
      }
    }

    let hcID = 'pending';
    let time = Date.now();

    if(typeof element.block !== 'undefined') {
      if (typeof element.block.hash !== 'undefined')
        hcID = element.block.hash;
      if (typeof element.block.timestamp !== 'undefined')
        time = element.block.timestamp;
    }

    hashcasts.push({
      msg: element.metadata,
      hcID,
      time,
    });

    //console.log("hashcasts:",hashcasts)

  });

  return hashcasts;
}

export const transactionsSlice = (page, PER_PAGE, transaction) => {
  const startingIndex = page === 1 ? 0 : ((page - 1) * PER_PAGE);
  const endingIndex = page * PER_PAGE;
  const paginatedTransactions = transaction.slice(startingIndex, endingIndex);
  return paginatedTransactions;
}

export const itemsSlice = (page, PER_PAGE, items) => {
  const startingIndex = page === 1 ? 0 : ((page - 1) * PER_PAGE);
  const endingIndex = page * PER_PAGE;
  const paginateditems = items.slice(startingIndex, endingIndex);
  return paginateditems;
}

export const bidsSlice = (page, PER_PAGE, bids) => {
  const startingIndex = page === 1 ? 0 : ((page - 1) * PER_PAGE);
  const result = Object.values(bids);
  let endingIndex = page * PER_PAGE;
  if(result.length < endingIndex) endingIndex = result.length;
  return result.slice(startingIndex, endingIndex);
}

export const hashSlice = (page, PER_PAGE, hashcasts) => {
  const startingIndex = page === 1 ? 0 : ((page - 1) * PER_PAGE);
  const result = Object.values(hashcasts);
  let endingIndex = page * PER_PAGE;
  if(result.length < endingIndex) endingIndex = result.length;
  return result.slice(startingIndex, endingIndex);
}

export const parseVarna = (transaction) => {

  const orderedTransactions = orderBy(transaction, i => i.block.timestamp, 'desc');

  const IDList = [];
  const swapMetaRaw = [];
  let isBeginner = true;

  orderedTransactions.forEach(element => {
    if (element.metadata.length === 32) {
      IDList.push(element.metadata);
      if(isBeginner) isBeginner = false; //this trips once, and then the beginner state is set
    }
    if (element.metadata.length === 32 && element.metadata.includes('-SWAP-')) {
      swapMetaRaw.push(element.metadata);
    }
  });

  return { 
    IDList, 
    orderedTransactions,
    swapMetaRaw,
    isBeginner,
  };
}