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

import EnyaFHE from 'enyafhe';
import { accMul } from 'util/calculation';
import { coinToArray } from 'util/coinConvert';

//Generate initial key set when the Seller lists their first item
export const generateItem = (
    itemToSend, 
    itemToSendAmount, 
    itemToSendAmountRemain, 
    itemToReceive, 
    sellerExchangeRate, 
    FHEseed, 
    cryptoWorkerThreadID
  ) => {

  if (typeof FHEseed === 'undefined') {
    console.log("export const generateItem: Sorry - no FHEseed defined");
    return
  }

  if (itemToSend.symbol.length < 3 || itemToSend.symbol.length > 4) {
    console.log("export const generateItem: Sorry - invalid symbol");
    return
  } 

  if (itemToReceive.symbol.length < 3 || itemToReceive.symbol.length > 4) {
    console.log("export const generateItem: Sorry - invalid symbol");
    return
  } 

  // generate key
  EnyaFHE.PrivateKeyGen(FHEseed);
  const fhePublicKey = EnyaFHE.PublicKeyGen();
  const fheMultiKey = EnyaFHE.MultiKeyGen();
  const fheRotaKey = EnyaFHE.RotaKeyGen();
  
  // Float to integer
  const itemToSendAmountInt = accMul(itemToSendAmount, Math.pow(10, 5));
  const sellerExchangeRateInt = accMul(sellerExchangeRate, Math.pow(10, 5));
  const itemToSendAmountRemainInt = accMul(itemToSendAmountRemain, Math.pow(10, 5));

  // Coin to array
  const itemToSendArray = coinToArray(itemToSend.symbol);
  const itemToReceiveArray = coinToArray(itemToReceive.symbol);

  // Vector to be encrypted
  // [tokenTosend, tokenToReceive, amountToSend, exchangeRate, remainAmountToSend]
  const vectorReady = [
    ...itemToSendArray, 
    ...itemToReceiveArray, 
    itemToSendAmountInt, 
    sellerExchangeRateInt, 
    itemToSendAmountRemainInt
  ];

  // Pack the vector
  const ptxt = EnyaFHE.PackVector(vectorReady);

  // Encrypt the vector
  const fheCiphertext = EnyaFHE.EncryptVector(
    ptxt,
    fhePublicKey
  );

  // post the result
  postMessage({
    fhePublicKey, 
    fheMultiKey, 
    fheRotaKey, 
    fheCiphertext, 
    cryptoWorkerThreadID, 
    status: 'success', 
    type: 'listItem'
  });

}

// Display item characteristics to the seller
export const decryptItem = (itemID, ciphertext, FHEseed) => {

  if (typeof FHEseed === 'undefined') {
    console.log("export const decryptItem: Sorry - no FHEseed defined");
    return
  }

  try {
    const itemCleartext = EnyaFHE.DecryptVector(
      EnyaFHE.ReadCiphertext(ciphertext),
      EnyaFHE.PrivateKeyGen(FHEseed),
    );
    
    // the last item in the list should be 0
    if (itemCleartext[511] === 0) {
      postMessage({itemCleartext, status: 'success', itemID, type: 'decryptAsk'});
    } else {
      console.log("Could not decrypt Item:",itemCleartext)
      postMessage({itemCleartext: {}, status: 'failure', itemID, type: 'decryptAsk'});
    }
  } catch (error) {
    postMessage({itemCleartext: {}, status: 'failure', itemID, type: 'decryptAsk'});
  }

}

// Encrypt the bid
export const encryptBid = (
  itemToReceive, 
  itemToReceiveAmount, 
  itemToSend, 
  buyerExchangeRate, 
  FHEseed, 
  cryptoWorkerThreadID
) => {

  if (typeof FHEseed === 'undefined') {
    console.log("export const generateItem: Sorry - no FHEseed defined");
    return
  }

  if (itemToReceive.symbol.length < 3 || itemToReceive.symbol.length > 4) {
    console.log("export const generateItem: Sorry - invalid symbol");
    return
  } 

  if (itemToSend.symbol.length < 3 || itemToSend.symbol.length > 4) {
    console.log("export const generateItem: Sorry - invalid symbol");
    return
  } 

  // FHE key
  EnyaFHE.PrivateKeyGen(FHEseed);
  const fhePublicKey = EnyaFHE.PublicKeyGen();  

  // Float to integer
  const itemToReceiveAmountInt = accMul(itemToReceiveAmount, Math.pow(10, 5));
  const buyerExchangeRateInt = accMul(buyerExchangeRate, Math.pow(10, 5));

  // Coin to array
  const itemToReceiveArray = coinToArray(itemToReceive.symbol);
  const itemToSendArray = coinToArray(itemToSend.symbol);

  // Vector to be encrypted
  const vectorReady = [
    ...itemToReceiveArray, 
    ...itemToSendArray, 
    itemToReceiveAmountInt, 
    buyerExchangeRateInt
  ];

  // Pack the vector
  const ptxt = EnyaFHE.PackVector(vectorReady);

  const bidCiphertext = EnyaFHE.EncryptVector(
    ptxt,
    fhePublicKey
  );
  
  postMessage({bidCiphertext, status: 'success', cryptoWorkerThreadID, type: 'encryptBid'});
}

// Send an offer to a seller
export const generateOffer = (bid, bidID, publicKey) => {

  const ptxt = EnyaFHE.PackVector([...bid]);
  
  const bidCiphertext = EnyaFHE.EncryptVector(
    ptxt,
    publicKey
  );

  postMessage({bidCiphertext, status: 'success', bidID, type: 'generateOffer'});
}

// Display bid characteristics to the buyer
export const decryptBid = (ciphertext, FHEseed, bidID) => {

  const fhePrivateKey = EnyaFHE.PrivateKeyGen(FHEseed);

  try {
    const bidCleartext = EnyaFHE.DecryptVector(
      EnyaFHE.ReadCiphertext(ciphertext),
      fhePrivateKey,
    );
    
    // the last element in the array should always be zero
    // if the decrypt succeeded
    if (bidCleartext[511] === 0) {
      postMessage({bidCleartext, status: 'success', bidID, type: 'decryptBid'});
    } else {
      postMessage({bidCleartext: {}, status: 'failure', bidID, type: 'decryptBid'});
    }
  } catch (error) {
    postMessage({bidCleartext: {}, status: 'failure', bidID, type: 'decryptBid'});
  }

}

// Generate AES key
export const generateAESKey = async (password, mode='AES-GCM', length='256') => {

  const algo = {
    name: 'PBKDF2',
    hash: 'SHA-256',
    salt: new TextEncoder().encode('a-unique-salt'),
    iterations: 1000
  };
  
  const derived = { name: mode, length: length };
  const encoded = new TextEncoder().encode(password);
  const key = await crypto.subtle.importKey('raw', encoded, { name: 'PBKDF2' }, false, ['deriveKey']);
  
  return crypto.subtle.deriveKey(algo, key, derived, false, ['encrypt', 'decrypt']);
}

export const AESEncrypt = async (text, key, mode='AES-GCM', length='256', ivLength='12') => {

  const algo = {
    name: mode,
    length: length,
    iv: crypto.getRandomValues(new Uint8Array(ivLength))
  };

  const encoded = new TextEncoder().encode(text);
  
  return {
    ciphertext: await crypto.subtle.encrypt(algo, key, encoded),
    iv: algo.iv
  };
}

export const AESDecrypt = async (encrypted, key, mode='AES-GCM', length='256') => {

  const algo = {
    name: mode,
    length: length,
    iv: encrypted.iv
  };

  const decrypted = await crypto.subtle.decrypt(algo, key, encrypted.ciphertext);
  
  return new TextDecoder().decode(decrypted);
}

export function hex (buff) {
  return [].map.call(new Uint8Array(buff), b => ('00' + b.toString(16)).slice(-2)).join('');
}
  
// Base64 encode
export function encode64 (buff) {
  return btoa(new Uint8Array(buff).reduce((s, b) => s + String.fromCharCode(b), ''));
}