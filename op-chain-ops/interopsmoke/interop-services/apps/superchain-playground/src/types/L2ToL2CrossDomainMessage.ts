import type { Address, Hex } from "viem";

export type L2ToL2CrossDomainMessage = {
  destination: bigint;
  messageNonce: bigint;
  sender: Address;
  target: Address;
  message: Hex;
};
