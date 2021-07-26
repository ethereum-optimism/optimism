## Network Configuration

This folder contains the configuration for given networks (e.g. `rinkeby.json` is the configuration for the Rinkeby test-net). These configuration files are meant to be used to configure external applications (like dApps) and thus contain a base set of information that may be useful (such as the address the Comptroller and a list of cToken markets). These configuration files are auto-generated when doing local development.

Structure
---------

```json
{
  "Contracts": {
    "MoneyMarket": "0x{address}",
    "Migrations": "0x{address}",
    "PriceOracle": "0x{address}",
    "InterestRateModel": "0x{address}"
  },
  "Tokens": {
    "{SYM}": {
      "name": "{Full Name}",
      "symbol": "{SYM}",
      "decimals": 18,
      "address": "0x{address}",
      "supported": true
    }
  }
}
```