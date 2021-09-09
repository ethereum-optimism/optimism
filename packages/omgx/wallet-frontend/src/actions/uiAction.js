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

export function setPage (page) {
  return function (dispatch) {
    return dispatch({ type: 'UI/PAGE/UPDATE', payload: page });
  }
}

export function setTheme (theme) {
  return function (dispatch) {
    return dispatch({ type: 'UI/THEME/UPDATE', payload: theme });
  }
}

export function openModal (modal, token, fast) {
  return function (dispatch) {
    return dispatch({ type: 'UI/MODAL/OPEN', payload: modal, token, fast });
  }
}

export function closeModal (modal) {
  return function (dispatch) {
    return dispatch({ type: 'UI/MODAL/CLOSE', payload: modal });
  }
}

export function openAlert (message) {
  return function (dispatch) {
    return dispatch({ type: 'UI/ALERT/UPDATE', payload: message });
  }
}

export function closeAlert () {
  return function (dispatch) {
    return dispatch({ type: 'UI/ALERT/UPDATE', payload: null });
  }
}

export function openError (message) {
  return function (dispatch) {
    return dispatch({ type: 'UI/ERROR/UPDATE', payload: message });
  }
}

export function closeError () {
  return function (dispatch) {
    return dispatch({ type: 'UI/ERROR/UPDATE', payload: null });
  }
}

export function ledgerConnect (derivation) {
  return function (dispatch) {
    return dispatch({ type: 'UI/LEDGER/UPDATE', payload: derivation });
  }
}

export function setActiveHistoryTab1 (tab) {
  return function (dispatch) {
    return dispatch({ type: 'UI/HISTORYTAB/UPDATE1', payload: tab });
  }
}

export function setActiveHistoryTab2 (tab) {
  return function (dispatch) {
    return dispatch({ type: 'UI/HISTORYTAB/UPDATE2', payload: tab });
  }
}

export function setModalData (modal, data) {
  return function (dispatch) {
    return dispatch({ type: 'UI/MODAL/DATA', payload: { modal, data } });
  }
}
