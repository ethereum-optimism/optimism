// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

type LocalAccount is address;
type RemoteAccount is address;

// Provides a cleaner syntax than the one solidity provides for custom types.
library TypedAddresses {
    function toAddress(LocalAccount self) internal pure returns (address) {
        return LocalAccount.unwrap(self);
    }

    function toAddress(RemoteAccount self) internal pure returns (address) {
        return RemoteAccount.unwrap(self);
    }

    function toLocal(address self) internal pure returns (LocalAccount) {
        return LocalAccount.wrap(self);
    }

    function toRemote(address self) internal pure returns (RemoteAccount) {
        return RemoteAccount.wrap(self);
    }
}
