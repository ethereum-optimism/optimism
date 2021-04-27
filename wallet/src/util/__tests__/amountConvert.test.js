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

import { logAmount, powAmount } from 'util/amountConvert';

describe('logAmount', () => {
  it('renders the highest denomination as a string', () => {
    const res = logAmount('1', 18);
    expect(res).toBe('0.000000000000000001');
    expect(typeof res).toBe('string');
  });

  it('can safely accept any amount or power type', () => {
    const res = logAmount(1, 18);
    expect(res).toBe('0.000000000000000001');

    const res2 = logAmount('1.0', '18');
    expect(res2).toBe('0.000000000000000001');
  });

  it('can handle unsafe numbers', () => {
    const res = logAmount('999999999999999999999999999999999', 18);
    expect(res).toBe('999999999999999.999999999999999999');
  });
});

describe('powAmount', () => {
  it('renders the lowest denomination as a string', () => {
    const res = powAmount('1', 18);
    expect(res).toBe('1000000000000000000');
    expect(typeof res).toBe('string');
  });

  it('can safely accept any amount or power type', () => {
    const res = powAmount(1, 18);
    expect(res).toBe('1000000000000000000');

    const res2 = powAmount('1.0', '18');
    expect(res2).toBe('1000000000000000000');
  });

  it('can handle unsafe numbers', () => {
    const res = powAmount('999999999999999999999999999999999', 18);
    expect(res).toBe('999999999999999999999999999999999000000000000000000');
  });
});
