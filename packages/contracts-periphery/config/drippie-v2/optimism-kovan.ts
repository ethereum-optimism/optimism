import { ethers } from 'ethers'

import { DrippieConfigV2 } from '../../src'
import { abi as TeleportrABI } from '../../artifacts/contracts/L1/TeleportrWithdrawer.sol/TeleportrWithdrawer.json'
import { StateType } from '../../src/config/drippie-v2/vm'

const GELATO_FUNDER = '0xc37f6a6c4AB335E20d10F034B90386E2fb70bbF5'
const TONY_FAUCET = '0xa8019d6F7bC3008a0a708A422f223Ccb21b61eAD'
const TELEPORTR = '0x8d12A4C82D88B6007F86e4e08B89e80A2BE029Cc'
const TEST_ACCOUNT = '0x063bE0Af9711a170BE4b07028b320C90705fec7C'

const config: DrippieConfigV2 = {
  SimpleBalance: {
    init: (vm) => {
      return [
        vm.state.last.set(0, StateType.UINT256),
        vm.state.interval.set(120, StateType.UINT256),
      ]
    },
    check: (vm) => {
      return vm.and(
        vm.lt(
          vm.add(
            vm.state.last.get(StateType.UINT256),
            vm.state.interval.get(StateType.UINT256)
          ),
          vm.timestamp()
        ),
        vm.lt(vm.balance(TEST_ACCOUNT), ethers.utils.parseEther('1'))
      )
    },
    actions: (vm) => {
      return [
        vm.state.last.set(vm.timestamp(), StateType.UINT256),
        vm.transfer(TEST_ACCOUNT, ethers.utils.parseEther('0.001')),
      ]
    },
  },
  GelatoBalance: {
    init: (vm) => {
      return [vm.state.interval.set(1, StateType.UINT256)]
    },
    check: (vm) => {
      return vm.and(
        vm.lt(
          vm.add(
            vm.state.last.get(StateType.UINT256),
            vm.state.interval.get(StateType.UINT256)
          ),
          vm.timestamp()
        ),
        vm.lt(vm.gelato.balance(GELATO_FUNDER), ethers.utils.parseEther('0.1'))
      )
    },
    actions: (vm) => {
      return [
        vm.state.last.set(vm.timestamp(), StateType.UINT256),
        vm.gelato.deposit(GELATO_FUNDER, ethers.utils.parseEther('1')),
      ]
    },
  },
  TonyOptimismKovanFaucet: {
    init: (vm) => {
      return [vm.state.interval.set(1, StateType.UINT256)]
    },
    check: (vm) => {
      return vm.and(
        vm.lt(
          vm.add(
            vm.state.last.get(StateType.UINT256),
            vm.state.interval.get(StateType.UINT256)
          ),
          vm.timestamp()
        ),
        vm.lt(vm.balance(TONY_FAUCET), ethers.utils.parseEther('20'))
      )
    },
    actions: (vm) => {
      return [
        vm.state.last.set(vm.timestamp(), StateType.UINT256),
        vm.transfer(TONY_FAUCET, ethers.utils.parseEther('100')),
      ]
    },
  },
  TeleportrWithdrawal: {
    init: (vm) => {
      return [vm.state.interval.set(1, StateType.UINT256)]
    },
    check: (vm) => {
      return vm.and(
        vm.lt(
          vm.add(
            vm.state.last.get(StateType.UINT256),
            vm.state.interval.get(StateType.UINT256)
          ),
          vm.timestamp()
        ),
        vm.gt(vm.balance(TELEPORTR), ethers.utils.parseEther('0.1'))
      )
    },
    actions: (vm) => {
      return [
        vm.state.last.set(vm.timestamp(), StateType.UINT256),
        () => {
          const bal = vm.balance(TELEPORTR)
          return [
            vm.contract(TELEPORTR, TeleportrABI).withdrawFromTeleportr(),
            vm.transfer(TELEPORTR, bal),
          ]
        },
      ]
    },
  },
}

export default config
