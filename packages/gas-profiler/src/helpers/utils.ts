import { ethers, Wallet, Contract } from 'ethers';
import { Ganache } from './ganache';
import { Provider } from 'ethers/providers';
import { Interface } from 'ethers/utils';

export interface Toolbox {
  provider: Provider;
  wallet: Wallet;
  ganache?: Ganache;
}

export const getToolbox = async (): Promise<Toolbox> => {
  const sk = '0x0123456789012345678901234567890123456789012345678901234567890123';

  const ganache = new Ganache({
    accounts: [
      {
        secretKey: sk,
        balance: ethers.utils.parseEther('100'),
      }
    ],
    gasLimit: 0x989680,
  });
  await ganache.start();

  const provider = new ethers.providers.JsonRpcProvider(`http://localhost:${ganache.port}`);
  const wallet = new ethers.Wallet(sk, provider);

  return {
    provider,
    wallet,
    ganache,
  };
};

export const getInterface = (contract: Contract): Interface => {
  return new ethers.utils.Interface(contract.interface.abi);
};