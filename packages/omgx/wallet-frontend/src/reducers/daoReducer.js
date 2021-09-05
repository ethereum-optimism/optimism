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
  balance: 0,
  votes: 0,
  proposalList: []
}

function daoReducer(state = initialState, action) {
  switch (action.type) {
    case 'BALANCE/DAO/GET/SUCCESS':

      const { balance } = action.payload;
      return { ...state, balance };

    case 'VOTES/DAO/GET/SUCCESS':
      const { votes } = action.payload;
      return { ...state, votes };

    case 'PROPOSAL/GET/SUCCESS':
      const { proposalList } = action.payload;
      return { ...state, proposalList };

    default:
      return state;
  }
}

export default daoReducer;