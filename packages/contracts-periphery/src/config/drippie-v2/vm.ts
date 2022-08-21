import * as weiroll from '@weiroll/weiroll.js'
import { ethers } from 'ethers'

import { contracts, asWeirollContract } from './contracts'

export type ReturnValue = ReturnType<weiroll.Planner['add']>

export enum StateType {
  BYTES32,
  UINT256,
}

export class DrippieVM {
  planner: weiroll.Planner

  constructor(private name: string) {
    this.planner = new weiroll.Planner()
  }

  compile(): {
    commands: string[]
    state: string[]
  } {
    return this.planner.plan()
  }

  assert(value: any): ReturnValue {
    return this.planner.add(contracts.Assert.t(value))
  }

  and(a: any, b: any): ReturnValue {
    return this.planner.add(contracts.Comparison.and(a, b))
  }

  lt(a: any, b: any): ReturnValue {
    return this.planner.add(contracts.Comparison.lt(a, b))
  }

  gt(a: any, b: any): ReturnValue {
    return this.planner.add(contracts.Comparison.gt(a, b))
  }

  add(a: any, b: any): ReturnValue {
    return this.planner.add(contracts.Math.add(a, b))
  }

  balance(address: any): ReturnValue {
    return this.planner.add(contracts.Ethereum.balance(address))
  }

  timestamp(): ReturnValue {
    return this.planner.add(contracts.Ethereum.timestamp())
  }

  transfer(address: any, amount: any): ReturnValue {
    return this.planner.add(
      contracts.Ethereum['transfer(address,uint256)'](address, amount)
    )
  }

  contract(address: any, abi: any): weiroll.Contract {
    return asWeirollContract(address, abi)
  }

  gelato = {
    deposit: (address: any, value: any): ReturnValue => {
      return this.planner.add(contracts.Gelato.deposit(address, value))
    },

    balance: (address: any): ReturnValue => {
      return this.planner.add(contracts.Gelato.balance(address))
    },
  }

  state: {
    [key: string]: {
      get: (type?: StateType) => ReturnValue
      set: (val: any, type?: StateType) => ReturnValue
    }
  } = new Proxy(
    {},
    {
      get: (target, key) => {
        const hashed = ethers.utils.solidityKeccak256(
          ['string'],
          [`${this.name}:${String(key)}`]
        )

        return {
          get: (type = StateType.BYTES32) => {
            let ret = this.planner.add(contracts.State.get(hashed))
            if (type === StateType.UINT256) {
              ret = this.planner.add(contracts.Coersion.toUint256(ret))
            }
            return ret
          },
          set: (val: any, type = StateType.BYTES32) => {
            let inp = val
            if (type === StateType.UINT256) {
              inp = this.planner.add(contracts.Coersion.toBytes32(val))
            }
            return this.planner.add(contracts.State.set(hashed, inp))
          },
        }
      },
    }
  )
}
