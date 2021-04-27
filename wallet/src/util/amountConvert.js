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

// we use BigNumber here for decimal support
import BigNumber from 'bignumber.js';

export function logAmount (amount, power, truncate = 0) {
  
  const x = new BigNumber(amount);
  const exp = new BigNumber(10).pow(power);

  const calculated = x.div(exp);

  if(truncate > 0)
  	return calculated.toFixed(truncate);
  else 
  	return calculated.toFixed();
}

export function powAmount (amount, power) {
  const x = new BigNumber(amount);
  const exp = new BigNumber(10).pow(power);

  const calculated = x.multipliedBy(exp);
  return calculated.toFixed(0);
}
