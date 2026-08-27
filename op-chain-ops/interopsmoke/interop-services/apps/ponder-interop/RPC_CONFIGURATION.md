# Runtime RPC Configuration

This implementation allows you to configure custom RPC URLs at runtime using environment variables, without modifying code.

## ✅ What Was Implemented

### 1. **Shared RPC Configuration Helper** (`src/config/rpcConfig.ts`)
- `getRpcUrl()` function with logging and basic validation
- Predefined environment variable names for all networks
- Warning for non-HTTP(S) URLs

### 2. **Updated Configuration Files**
All ponder configs now support runtime RPC configuration:
- `ponder.config.alphanet.ts` - InteropRc networks
- `ponder.config.devnet.ts` - Interop networks
- `ponder.config.supersim.ts` - Supersim networks
- `ponder.config.v0.ts` - Interop v0 networks
- `ponder.config.jnt.v0.ts` - Interop JNT v0 networks

### 3. **Environment Variables**

| Network Config | Environment Variables |
|----------------|----------------------|
| **alphanet** | `INTEROP_RC_ALPHA0_RPC_URL`<br>`INTEROP_RC_ALPHA1_RPC_URL` |
| **devnet** | `INTEROP_ALPHA0_RPC_URL`<br>`INTEROP_ALPHA1_RPC_URL` |
| **supersim** | `SUPERSIM_L2A_RPC_URL`<br>`SUPERSIM_L2B_RPC_URL` |
| **v0** | `INTEROP_V0_0_RPC_URL`<br>`INTEROP_V0_1_RPC_URL` |
| **jnt.v0** | `INTEROP_JNT_V0_0_RPC_URL`<br>`INTEROP_JNT_V0_1_RPC_URL` |

### 4. **Example Configuration** (`.env.local.example`)
Complete example showing all available environment variables

## 🚀 Usage Examples

### Using .env.local file:
```bash
# Create .env.local
cp .env.local.example .env.local
# Edit with your RPC URLs
pnpm dev:alphanet
```

### Inline environment variables:
```bash
INTEROP_V0_0_RPC_URL=https://your-rpc.com \
INTEROP_V0_1_RPC_URL=https://your-rpc2.com \
pnpm dev:v0
```

### Local development with Supersim:
```bash
SUPERSIM_L2A_RPC_URL=http://localhost:9545 \
SUPERSIM_L2B_RPC_URL=http://localhost:9546 \
pnpm dev:supersim
```

## 📋 Features

✅ **Fallback Support**: Uses default chain RPC URLs if env vars not set
✅ **Logging**: Clear indication of which RPC URLs are being used
✅ **Validation**: Basic URL format checking with warnings
✅ **All Networks**: Supports alphanet, devnet, and supersim configs
✅ **No Code Changes**: Runtime configuration without touching source code

## 🔍 Verification

When starting Ponder, you'll see logs like:
```
📡 Using custom RPC for InteropRcAlpha0: https://your-custom-rpc.com
📡 Using default RPC for InteropRcAlpha1: https://interop-rc-alpha-1.optimism.io

📋 Contract Configuration:
📮 L2ToL2CrossDomainMessenger: 0x4200000000000000000000000000000000000023
⛽ Gas Tank Contract: 0x0987654321098765432109876543210987654321
🤝 Promise Contract: 0x1234567890123456789012345678901234567890
```

This confirms which RPC URLs are being used for each network and which contract addresses are configured for indexing.

## 📝 Contract Address Configuration

The system automatically logs all configured contract addresses on startup:

### Environment Variables for Contract Addresses
- `PROMISE_CONTRACT_ADDRESS` - Address of the Promise contract to index
- `GAS_TANK_CONTRACT_ADDRESS` - Address of the Gas Tank contract to index

### Contract Address Logging
- **L2ToL2CrossDomainMessenger**: Always uses the standard predefined address `0x4200000000000000000000000000000000000023`
- **Gas Tank Contract**: Set via `GAS_TANK_CONTRACT_ADDRESS` environment variable
- **Promise Contract**: Set via `PROMISE_CONTRACT_ADDRESS` environment variable

### Configuration Status
If a contract address is not configured, you'll see a warning:
```
🤝 Promise Contract: ❌ NOT CONFIGURED (set PROMISE_CONTRACT_ADDRESS)
⛽ Gas Tank Contract: ❌ NOT CONFIGURED (set GAS_TANK_CONTRACT_ADDRESS)
```

If properly configured, you'll see the actual addresses:
```
🤝 Promise Contract: 0x1234567890123456789012345678901234567890
⛽ Gas Tank Contract: 0x0987654321098765432109876543210987654321
```
