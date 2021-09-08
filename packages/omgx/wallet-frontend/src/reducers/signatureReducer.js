/*
Copyright 2019-present OmiseGO Pte Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License. */

const initialState = {
  exitLPsigned: false,
  exitTRADsigned: false,
  depositLPsigned: false,
  depositTRADsigned: false
}

function signatureReducer (state = initialState, action) {
  switch (action.type) {
    case 'EXIT/LP/SIGNED':
      console.log('exitLPsigned:',action.payload)
      return {
        ...state,
        exitLPsigned: action.payload
      }
      case 'EXIT/TRAD/SIGNED':
      console.log('exitTRADsigned:',action.payload)
      return {
        ...state,
        exitTRADsigned: action.payload
      }
      case 'DEPOSIT/LP/SIGNED':
      console.log('depositLPsigned:',action.payload)
      return {
        ...state,
        depositLPsigned: action.payload
      }
      case 'DEPOSIT/TRAD/SIGNED':
      console.log('depositTRADsigned:',action.payload)
      return {
        ...state,
        depositTRADsigned: action.payload
      }
    default:
      return state;
  }
}

export default signatureReducer;
