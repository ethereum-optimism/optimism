# Bridge Example

Example of on-ramp and off-ramp for BOBA and ETH between L1 and L2.

## Update .env

Add a `.env` to `/boba-examples/init-fund-l2`. You will need to provide your private key.

```
L1_NODE_URL=https://sepolia.gateway.tenderly.co
PRIVATE_KEY=
```

## Install packages

After adding the `.env`, run the following command to install packages

```bash
yarn install
```

## Bridge tokens

To bridge ETH from L1 to L2, run

```bash
yarn deposit:ETH
```

To bridge BOBA from L1 to L2, run

```bash
yarn deposit:BOBA
```

To exit ETH from L2 to L1, run

```bash
yarn withdraw:ETH
```

To exit BOBA from L2 to L1, run

```bash
yarn withdraw:BOBA
```

To bridge ETH and then withdraw it, run

```bash
yarn start
```



