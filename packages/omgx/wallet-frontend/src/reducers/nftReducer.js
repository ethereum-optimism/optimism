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

//localStorage.removeItem("nftContracts")

let nftContracts = localStorage.getItem("nftContracts")

if (nftContracts) {
  nftContracts = JSON.parse(nftContracts)
  console.log("NFT Contracts Cache:",nftContracts)
}

let nftFactories = localStorage.getItem("nftFactories")

if (nftFactories) {
  nftFactories = JSON.parse(nftFactories)
  console.log("NFT Factories Cache:",nftFactories)
}

let nftList = localStorage.getItem("nftList")

if (nftList) {
  nftList = JSON.parse(nftList)
  console.log("NFT List Cache:",nftList)
}


const initialState = {
  list: nftList ? nftList : {},
  factories: nftFactories ? nftFactories : {},
  contracts: nftContracts ? nftContracts : {}
}

function nftReducer (state = initialState, action) {
  switch (action.type) {
    
    case 'NFT/GET/SUCCESS':

      localStorage.setItem("nftList", JSON.stringify({
          ...state.list,
          [action.payload.UUID]: action.payload
        })
      )

      return { 
        ...state,
        list: {
          ...state.list,
          [action.payload.UUID]: action.payload
        } 
      }

    case 'NFT/ADDCONTRACT/SUCCESS':

      const address = action.payload

      localStorage.setItem("nftContracts", JSON.stringify({
          ...state.contracts,
          [address]: address
        })
      )

      return { 
        ...state,
        contracts: {
          ...state.contracts,
          [address]: address
        }
      }

    case 'NFT/CREATEFACTORY/SUCCESS':

      localStorage.setItem("nftFactories", JSON.stringify({
          ...state.factories,
          [action.payload.address]: action.payload
        })
      )

      return { 
        ...state,
        factories: {
          ...state.factories,
          [action.payload.address]: action.payload
        }
      }
      
    default:
      return state;
  }
}

export default nftReducer;
