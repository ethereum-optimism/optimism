/**
 *
 */

// This file is copied from ethers because optimism may not be available on
// every testnet. The networks that optimism is available on should be added to
// the networks list below.

import { Network, Networkish } from "@ethersproject/networks";
import { Logger } from "@ethersproject/logger";
import { ConnectionInfo } from "@ethersproject/web";
import { isUrl } from './utils';

import pkg = require('../../package.json')

const version = pkg.version
const logger = new Logger(version);

type DefaultProviderFunc = (providers: any, options?: any) => any;

interface Renetworkable extends DefaultProviderFunc {
    renetwork: (network: Network) => DefaultProviderFunc;
};

function isRenetworkable(value: any): value is Renetworkable {
    return (value && typeof(value.renetwork) === "function");
}

export const homestead: Network = {
    chainId: 1,
    ensAddress: null,
    name: "homestead",
    _defaultProvider: null,
};

// TODO(mark): add each supported Network to this list
const networks = [
  homestead
]

/**
 *  getNetwork
 *
 *  Converts a named common networks or chain ID (network ID) to a Network
 *  and verifies a network is a valid Network..
 */
export function getNetwork(network: Networkish): Network {
    // No network (null)
    if (network == null) { return null; }

    if (typeof(network) === "number") {
        for (const name of Object.keys(networks)) {
            // tslint:disable-next-line:no-shadowed-variable
            const standard = networks[name];
            if (standard.chainId === network) {
                return {
                    name: standard.name,
                    chainId: standard.chainId,
                    ensAddress: (standard.ensAddress || null),
                    _defaultProvider: (standard._defaultProvider || null)
                };
            }
        }

        return {
            chainId: network,
            name: "unknown"
        };
    }

    if (typeof(network) === "string") {
        // tslint:disable-next-line:no-shadowed-variable
        const standard = networks[network];
        if (standard == null) { return null; }
        return {
            name: standard.name,
            chainId: standard.chainId,
            ensAddress: standard.ensAddress,
            _defaultProvider: (standard._defaultProvider || null)
        };
    }

    const standard  = networks[network.name];

    // Not a standard network; check that it is a valid network in general
    if (!standard) {
        if (typeof(network.chainId) !== "number") {
            logger.throwArgumentError("invalid network chainId", "network", network);
        }
        return network;
    }

    // Make sure the chainId matches the expected network chainId (or is 0; disable EIP-155)
    if (network.chainId !== 0 && network.chainId !== standard.chainId) {
        logger.throwArgumentError("network chainId mismatch", "network", network);
    }

    // @TODO: In the next major version add an attach function to a defaultProvider
    // class and move the _defaultProvider internal to this file (extend Network)
    let defaultProvider: DefaultProviderFunc = network._defaultProvider || null;
    if (defaultProvider == null && standard._defaultProvider) {
        if(isRenetworkable(standard._defaultProvider)) {
            defaultProvider = standard._defaultProvider.renetwork(network);
        } else {
            defaultProvider = standard._defaultProvider;
        }
    }

    // Standard Network (allow overriding the ENS address)
    return {
        name: network.name,
        chainId: standard.chainId,
        ensAddress: (network.ensAddress || standard.ensAddress || null),
        _defaultProvider: defaultProvider
    };
}

// Based on the newtork, return the public URL of the optimism nodes
// TODO(mark): add public urls here
export function getUrl(network: Network, extra: Networkish): string | ConnectionInfo {
  let host: string = null

  // Allow for custom urls to be passed in
  if (typeof extra === 'string' && isUrl(extra)) {
    return { url: extra }
  }

  // List of publically available urls to use
  // TODO(mark): in this case, turn off calls for `eth_getChainId`
  switch (network ? network.name : 'unknown') {
    case 'main':
      host = '' // TODO: once the url of mainnet is known
    break
    default:
      logger.throwError("unsupported network", Logger.errors.INVALID_ARGUMENT, {
      argument: "network",
      value: network
    });
  }

  const connection: ConnectionInfo = {
    url: `http://${host}`
  };

  return connection
}

