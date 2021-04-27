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

export function selectModalState (modal) {
  return function (state) {
    return state.ui[modal];
  };
}

export function selectAlert (state) {
  return state.ui.alert;
}

export function selectError (state) {
  return state.ui.error;
}

export function selectLedger (state) {
  return state.ui.ledger;
}

export function selectActiveHistoryTab (state) {
  return state.ui.activeHistoryTab;
}
