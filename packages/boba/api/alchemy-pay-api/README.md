# Alchemy Pay URL Generator Service

A simple serverless service to generate signed URLs for Alchemy Pay integration.

## Prerequisites

- Node.js (v14 or later)
- Python 3.9
- AWS CLI configured with appropriate credentials
- Serverless Framework

## Quick Start

1. **Install Dependencies**
```bash
# Install Serverless Framework globally
npm install -g serverless

# Install project dependencies
pnpm install

# Install Python dependencies
pip install -r requirements.txt
```

2. **Configure Environment**
```bash
# Copy example environment files
cp env-dev.example.yml env-dev.yml
cp env-mainnet.example.yml env-mainnet.yml

# Edit the files with your credentials
```

3. **Local Testing**
```bash
# Start local serverless offline
pnpm run dev

# Test URL generation
curl -X POST http://localhost:3000/dev/generate \
  -H "Content-Type: application/json" \
  -d '{
    "crypto": "USDT",
    "fiatAmount": "15",
    "fiat": "USD",
    "merchantOrderNo": "test123",
    "network": "BSC"
  }'
```

4. **Deployment**
```bash
# Deploy to dev
./deploy.sh dev

# Deploy to mainnet (requires MAINNET_SECRET_KEY env variable)
./deploy.sh mainnet
```

## Project Structure

```
api/
├── src/
│   └── handler.py      # Main Lambda handler
├── test/
│   └── test_handler.py # Unit tests
├── serverless.yml      # Main serverless config
├── env-dev.yml         # Dev environment variables
├── env-mainnet.yml     # Mainnet environment variables
├── requirements.txt    # Python dependencies
├── package.json        # Node.js dependencies
└── deploy.sh          # Deployment script
```

## Environment Variables

Required environment variables in `env-*.yml`:

- `APP_ID`: Your Alchemy Pay App ID
- `SECRET_KEY`: Your Alchemy Pay Secret Key
- `BASE_URL`: API base URL (dev/mainnet)
- `STAGE`: Deployment stage

## Local Development

1. Install dependencies:
```bash
pnpm install
pip install -r requirements.txt
```

2. Start local server:
```bash
pnpm run dev
```

3. Test the endpoint:
```bash
curl -X POST http://localhost:3000/dev/generate \
  -H "Content-Type: application/json" \
  -d '{
    "crypto": "USDT",
    "fiatAmount": "15",
    "fiat": "USD",
    "merchantOrderNo": "test123",
    "network": "BSC",
    "callbackUrl": "https://your-callback.com",
    "redirectUrl": "https://your-redirect.com"
  }'
```
