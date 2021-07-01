# Liquidity Pool

<img width="1243" alt="LP" src="https://user-images.githubusercontent.com/46272347/119060612-6455cc00-b987-11eb-9f8c-dfadfa029951.png">

The L2 liquidity pool is the main pool. It provides a way to deposit and withdraw tokens for liquidity providers. Swap users can deposit ETH or ERC20 tokens to fastly exit the L2.

The L1 liquidity pool is the sub pool. Swap users can do fast onramp. When swap users do a fast exit via the L2 liquidity pool, it sends funds to the swap users.

For OMGX, there are no delays for users to move funds from L1 to L2. The liquidity pool is used to help users quickly exit L2.

## Calculation

* A deposit 100

  **A info**

  | Deposit Amount | Reward Debet | Pending Reward |
  | -------------- | ------------ | -------------- |
  | 100            | 0            | 0              |

  **Pool info**

  | Total Rewards | Reward Per Share | Total Deposit Amount |
  | ------------- | ---------------- | -------------------- |
  | 0             | 0                | 100                  |

* The pool generates 10 rewards

  **Pool info**

  | Total Rewards | Reward Per Share | Total Deposit Amount |
  | ------------- | ---------------- | -------------------- |
  | 10            | 0                | 100                  |

* B deposit 100

  We need to update the rewardPerShare first (don't consider the new deposit amount first!)

  **Pool info**

  | Total Rewards | Reward Per Share | Total Deposit Amount |
  | ------------- | ---------------- | -------------------- |
  | 10            | 10 / 100         | 100                  |

  Calculate the B info

  **B info**

  | Deposit Amount | Reward Debet                                        | Pending Reward |
  | -------------- | --------------------------------------------------- | -------------- |
  | 100            | rewardPerShare * depositAmount = 100 * 10/ 100 = 10 | 0              |

  The total deposit amount of the pool is 200 now.

  **pool info**

  | Total Rewards | Reward Per Share | Total Deposit Amount |
  | ------------- | ---------------- | -------------------- |
  | 10            | 10 / 100         | 200                  |

* The pool generates another 5 rewards

  **Pool info**

  | Total Rewards | Reward Per Share | Total Deposit Amount |
  | ------------- | ---------------- | -------------------- |
  | 15            | 10/100           | 200                  |

* If A withdraw 100 tokens

  We need to update the rewardPerShare first.

  **Pool info**

  | Total Rewards | Reward Per Share                                             | Total Deposit Amount |
  | ------------- | ------------------------------------------------------------ | -------------------- |
  | 15            | 10 / 100 + (increased_rewards) / total_deposit_amount = 10 / 100 + 5 / 200 | 200                  |

  The rewards for A is 

  ```
  deposit_amount * reward_per_share - reward_debet = 100 * (10 / 100 + 5 / 200 ) - 0 = 12.5
  ```

* If B withdraw 100 tokens

  We need to update the rewardPerShare first.

  **Pool info**

  | Total Rewards | Reward Per Share                                             | Total Deposit Amount |
  | ------------- | ------------------------------------------------------------ | -------------------- |
  | 15            | 10 / 100 + (increased_rewards) / total_deposit_amount = 10 / 100 + 5 / 200 | 200                  |

  The rewards for B is

  ```
  deposit_amount * reward_per_share - reward_debet = 100 * (10 / 100 + 5 / 200 ) - 10 = 2.5
  ```

  