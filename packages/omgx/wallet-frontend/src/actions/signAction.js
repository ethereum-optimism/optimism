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

import store from 'store';

export async function updateSignatureStatus_exitLP ( sigStatus ) {
  store.dispatch({type: 'EXIT/LP/SIGNED',payload: sigStatus})
}

export async function updateSignatureStatus_exitTRAD ( sigStatus ) {
  store.dispatch({type: 'EXIT/TRAD/SIGNED',payload: sigStatus})
}

export async function updateSignatureStatus_depositLP ( sigStatus ) {
  store.dispatch({type: 'DEPOSIT/LP/SIGNED',payload: sigStatus})
}

export async function updateSignatureStatus_depositTRAD ( sigStatus ) {
  store.dispatch({type: 'DEPOSIT/TRAD/SIGNED',payload: sigStatus})
}
