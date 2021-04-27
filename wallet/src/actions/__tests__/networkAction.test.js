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

import * as actions from 'actions/networkAction';
import networkService from 'services/networkService';
import store from 'store';

jest.mock('services/networkService');
jest.mock('store');

describe('networkActions', () => {
  beforeEach(() => {
    store.clearActions();
    jest.clearAllMocks();
  });

  it('should dispatch correct actions on checkWatcherStatus', async () => {
    const expectedActions = [
      { type: 'STATUS/GET/REQUEST' },
      { type: 'STATUS/GET/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.checkWatcherStatus());
    expect(networkService.checkStatus).toHaveBeenCalled();
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on fetchBalances', async () => {
    const expectedActions = [
      { type: 'BALANCE/GET/REQUEST' },
      { type: 'BALANCE/GET/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.fetchBalances());
    expect(networkService.getBalances).toHaveBeenCalled();
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on fetchTransactions', async () => {
    const expectedActions = [
      { type: 'TRANSACTION/GETALL/REQUEST' },
      { type: 'TRANSACTION/GETALL/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.fetchTransactions());
    expect(networkService.getAllTransactions).toHaveBeenCalled();
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on fetchDeposits', async () => {
    const expectedActions = [
      { type: 'DEPOSIT/GETALL/REQUEST' },
      { type: 'DEPOSIT/GETALL/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.fetchDeposits());
    expect(networkService.getDeposits).toHaveBeenCalled();
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on fetchExits', async () => {
    const expectedActions = [
      { type: 'EXIT/GETALL/REQUEST' },
      { type: 'EXIT/GETALL/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.fetchExits());
    expect(networkService.getExits).toHaveBeenCalled();
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on checkForExitQueue', async () => {
    const expectedActions = [
      { type: 'QUEUE/GET_0x/REQUEST' },
      { type: 'QUEUE/GET/SUCCESS', payload: 'toto' },
      { type: 'QUEUE/GET_0x/SUCCESS' }
    ];
    const res = await store.dispatch(actions.checkForExitQueue('0x'));
    expect(res).toBe(true);
    expect(networkService.checkForExitQueue).toHaveBeenCalledWith('0x');
    expect(networkService.getExitQueue).toHaveBeenCalledWith('0x');
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on getExitQueue', async () => {
    const expectedActions = [
      { type: 'QUEUE/GET/REQUEST' },
      { type: 'QUEUE/GET/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.getExitQueue('0x'));
    expect(networkService.getExitQueue).toHaveBeenCalledWith('0x');
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on addExitQueue', async () => {
    const expectedActions = [
      { type: 'QUEUE/CREATE/REQUEST' },
      { type: 'QUEUE/CREATE/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.addExitQueue('0x', 1));
    expect(networkService.addExitQueue).toHaveBeenCalledWith('0x', 1);
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on exitUtxo', async () => {
    const expectedActions = [
      { type: 'EXIT/CREATE/REQUEST' },
      { type: 'EXIT/CREATE/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.exitUtxo({ foo: 'bar' }, 1));
    expect(networkService.exitUtxo).toHaveBeenCalledWith({ foo: 'bar' }, 1);
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on deposit', async () => {
    const expectedActions = [
      { type: 'DEPOSIT/CREATE/REQUEST' },
      { type: 'DEPOSIT/CREATE/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.depositEth(1, 1));
    expect(networkService.depositEth).toHaveBeenCalledWith(1, 1);
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on processExits', async () => {
    const expectedActions = [
      { type: 'QUEUE/PROCESS/REQUEST' },
      { type: 'QUEUE/PROCESS/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.processExits(1, '0x', 1));
    expect(networkService.processExits).toHaveBeenCalledWith(1, '0x', 1);
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on transfer', async () => {
    const expectedActions = [
      { type: 'TRANSFER/CREATE/REQUEST' },
      { type: 'TRANSFER/CREATE/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.transfer('data'));
    expect(networkService.transfer).toHaveBeenCalledWith('data');
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on mergeUtxos', async () => {
    const expectedActions = [
      { type: 'TRANSFER/CREATE/REQUEST' },
      { type: 'TRANSFER/CREATE/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.mergeUtxos(false, 'data'));
    expect(networkService.mergeUtxos).toHaveBeenCalledWith(false, 'data');
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on fetchGas', async () => {
    const expectedActions = [
      { type: 'GAS/GET/REQUEST' },
      { type: 'GAS/GET/SUCCESS', payload: 'toto' }
    ];
    await store.dispatch(actions.fetchGas());
    expect(networkService.getGasPrice).toHaveBeenCalled();
    expect(store.getActions()).toEqual(expectedActions);
  });

  it('should dispatch correct actions on fetchFees', async () => {
    const expectedActions = [
      { type: 'FEE/GET/REQUEST' },
      { type: 'FEE/GET/SUCCESS', payload: [ 1,2,3 ] }
    ];
    await store.dispatch(actions.fetchFees());
    expect(networkService.fetchFees).toHaveBeenCalled();
    expect(store.getActions()).toEqual(expectedActions);
  });
});
