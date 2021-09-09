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
  theme: 'dark',
  page: 'AccountNow',
  depositModal: false,
  transferModal: false,
  exitModal: false,
  mergeModal: false,
  confirmationModal: false,
  wrongNetworkModal: false,
  ledgerConnectModal: false,
  addNewTokenModal: false,
  farmDepositModal: false,
  farmWithdrawModal: false,
  transferDaoModal: false,
  delegateDaoModal: false,
  newProposalModal: false,
  ledger: false,
  alert: null,
  error: null,
  activeHistoryTab1: 'All',
  activeHistoryTab2: 'Exits',
};

function uiReducer (state = initialState, action) {
  switch (action.type) {
    case 'UI/THEME/UPDATE':
      return { ...state, theme: action.payload }
    case 'UI/PAGE/UPDATE':
      return { ...state, page: action.payload }
    case 'UI/MODAL/OPEN':
      return { ...state,
        [action.payload]: true,
        fast: action.fast,
        token: action.token
      }
    case 'UI/MODAL/CLOSE':
      return { ...state, [action.payload]: false }
    case 'UI/ALERT/UPDATE':
      return { ...state, alert: action.payload }
    case 'UI/ERROR/UPDATE':
      return { ...state, error: action.payload }
    case 'UI/LEDGER/UPDATE':
      return { ...state, ledger: action.payload }
    case 'UI/HISTORYTAB/UPDATE1':
      return { ...state, activeHistoryTab1: action.payload }
    case 'UI/HISTORYTAB/UPDATE2':
      return { ...state, activeHistoryTab2: action.payload }
    case 'UI/MODAL/DATA':
      let dataType = 'generic';
      if(action.payload.modal === 'confirmationModal') {
        dataType = 'cMD';
      }
      return { ...state,
        [dataType]: action.payload.data,
      }
    default:
      return state;
  }
}

export default uiReducer;
