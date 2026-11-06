# L2 Rollup 생성 - 코드 예제

이 디렉터리에는 [L2 Rollup 생성 튜토리얼](/ko/chain-operators/tutorials/create-l2-rollup)과 함께 제공되는 완전한 동작 구현이 들어 있습니다. 공식 배포된 Docker 이미지를 사용하여 OP Stack L2 rollup testnet을 자동으로 배포합니다.

## 개요

이 구현은 다음을 포함하여 완전히 동작하는 OP Stack L2 rollup testnet을 배포합니다.

- Sepolia testnet에 배포되는 **L1 스마트 컨트랙트**(op-deployer 사용)
- 트랜잭션을 처리하는 **실행 클라이언트**(op-reth)
- rollup 합의를 관리하는 **합의 클라이언트**(op-node)
- 트랜잭션 데이터를 L1에 게시하는 **Batcher**(op-batcher)
- state root 제안을 제출하는 **Proposer**(op-proposer)
- 분쟁을 모니터링하는 **Challenger**(op-challenger)

## 사전 요구 사항

### 소프트웨어 의존성

| 의존성 | 버전 | 설치 명령 | 용도 |
|------------|---------|-----------------|---------|
| [Docker](https://docs.docker.com/get-docker/) | ^24 | Docker 설치 가이드를 따르십시오 | OP Stack 서비스를 위한 컨테이너 런타임 |
| [Docker Compose](https://docs.docker.com/compose/install/) | latest | 일반적으로 Docker Desktop에 포함 | 멀티 컨테이너 오케스트레이션 |
| [jq](https://github.com/jqlang/jq) | latest | `brew install jq` (macOS) / `apt install jq` (Ubuntu) | 배포 데이터를 위한 JSON 처리 |
| [git](https://git-scm.com/) | latest | 일반적으로 사전 설치됨 | prestate 생성을 위한 저장소 복제 |

### 권장: 도구 관리

올바른 도구 버전으로 최상의 경험을 얻으려면 [mise](https://mise.jdx.dev/)를 설치하는 것을 권장합니다.

```bash
# Install mise (manages all tool versions automatically)
curl https://mise.jdx.dev/install.sh | bash

# Install all required tools with correct versions
cd docs/create-l2-rollup-example
mise install
```

**왜 mise인가요?** 도구 설치와 버전 관리를 자동으로 처리하여 호환성 문제를 방지합니다.

### 네트워크 접근

- **Sepolia RPC URL**: [Infura](https://infura.io), [Alchemy](https://alchemy.com) 또는 기타 제공업체에서 발급받으십시오.
- **Sepolia ETH**: 컨트랙트 배포와 운영을 위해 최소 2~3 ETH가 필요합니다.
- **공인 IP**: P2P 네트워킹에 필요합니다(`curl ifconfig.me`로 공인 IP를 확인할 수 있습니다).

## 빠른 시작

1. **이 코드 디렉터리로 이동합니다**:
   ```bash
   cd docs/create-l2-rollup-example
   ```

2. **환경 변수를 구성합니다**:
   ```bash
   cp .example.env .env
   # Edit .env with your actual values (L1_RPC_URL, PRIVATE_KEY, L2_CHAIN_ID)
   ```

3. **op-deployer를 다운로드합니다**:
   ```bash
   make init    # Download op-deployer
   ```

4. **모든 것을 배포하고 시작합니다**:
   ```bash
   make setup   # Deploy contracts and configure all services
   make up      # Start all services
   ```

5. **배포를 확인합니다**:
   ```bash
   make status  # Check service health
   make test-l1 # Verify L1 connectivity
   make test-l2 # Verify L2 functionality

   # Or manually check:
   docker-compose ps
   docker-compose logs -f op-node
   ```

## 환경 구성

`.example.env`를 `.env`로 복사하고 다음 변수를 구성합니다.

```bash
# L1 Configuration (Sepolia testnet)
# Option 1: Public endpoint (no API key required)
L1_RPC_URL="https://ethereum-sepolia-rpc.publicnode.com"
L1_BEACON_URL="https://ethereum-sepolia-beacon-api.publicnode.com"

# Option 2: Private endpoint (requires API key)
# L1_RPC_URL="https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY"
# L1_BEACON_URL="https://ethereum-sepolia-beacon-api.publicnode.com"

# Private key for deployment and operations
# IMPORTANT: Never commit this to version control
PRIVATE_KEY="YOUR_PRIVATE_KEY_WITHOUT_0x_PREFIX"

# Optional: Public IP for P2P networking (defaults to 127.0.0.1 for local testing)
P2P_ADVERTISE_IP="127.0.0.1"

# Optional: Custom L2 Chain ID (default: 16584)
L2_CHAIN_ID="16584"
```

`.env` 파일은 Docker Compose가 자동으로 로드합니다.

## 수동 설정(대안)

자세한 수동 설정 방법은 [L2 Rollup 생성 튜토리얼](/ko/chain-operators/tutorials/create-l2-rollup)을 참조하십시오. 자동화된 방식을 사용하지 않으려는 경우, 이 튜토리얼에서 각 구성 요소를 개별적으로 설정하는 단계별 안내를 제공합니다.

## 디렉터리 구조

```
create-l2-rollup-example/
├── .example.env           # Environment variables template
├── docker-compose.yml     # Service orchestration
├── Makefile               # Automation commands
├── scripts/
│   ├── setup-rollup.sh   # Automated deployment script
│   └── download-op-deployer.sh # op-deployer downloader
└── README.md             # This file
```

**생성되는 디렉터리**(배포 중에 생성됨):

```
deployer/                # op-deployer configuration and outputs
├── .deployer/           # Deployment artifacts (genesis.json, rollup.json, etc.)
├── addresses/           # Generated wallet addresses
└── .env                 # Environment variables
batcher/                 # op-batcher configuration
└── .env                 # Environment variables
proposer/                # op-proposer configuration
└── .env                 # Environment variables
challenger/              # op-challenger configuration
├── .env                 # Challenger-specific environment variables
└── data/                # Challenger data directory
sequencer/               # op-sequencer configuration
├── .env                 # op-sequencer environment variables
├── genesis.json         # L2 genesis file (initializes op-reth)
├── jwt.txt              # JWT secret for auth RPC
├── rollup.json          # op-node rollup configuration
└── op-reth-data/        # op-reth data directory (Docker named volume)
```

## 서비스 포트

| 서비스 | 포트 | 설명 |
|---------|------|-------------|
| op-reth | 8545 | HTTP RPC 엔드포인트 |
| op-reth | 8546 | WebSocket RPC 엔드포인트 |
| op-reth | 8551 | op-node용 Auth RPC |
| op-node | 8547 | op-node RPC 엔드포인트 |
| op-node | 9222 | P2P 네트워킹 |

## 모니터링 및 로그

```bash
# View all service logs
make logs

# View specific service logs
docker-compose logs -f op-node
docker-compose logs -f op-reth

# Check service health
make status

# Restart all services
make restart

# Restart a specific service
docker-compose restart op-batcher
```

## P2P 네트워킹 구성

기본적으로 이 devnet은 로컬에서 실행할 때 검증 경고가 발생하지 않도록 P2P 네트워킹을 완전히 비활성화합니다. `docker-compose.yml`의 `op-node` 서비스에 `--p2p.disable` 플래그가 설정되어 있습니다.

<Warning>
**프로덕션 배포의 경우**, `--p2p.disable` 플래그를 반드시 **제거**하고 P2P 네트워킹을 올바르게 구성해야 합니다. P2P는 다음을 위해 필수적입니다.
- 새로 시퀀싱된 블록을 네트워크의 다른 node에 배포
- 피어 node가 체인을 동기화하고 검증할 수 있도록 지원
- 탈중앙화된 node 네트워크 지원
- 네트워크 복원력과 이중화
</Warning>

### P2P를 활성화해야 하는 경우

| 환경 | P2P 네트워킹 | 이유 |
|-------------|---------------|---------|
| **로컬 devnet** | 비활성화(기본값) | 피어 없이 단독으로 테스트할 때 P2P 경고 방지 |
| **비공개 testnet** | 비활성화 | 연결할 다른 node가 없음 |
| **공개 testnet** | 활성화 | 다른 node가 블록을 수신하고 동기화해야 함 |
| **프로덕션 mainnet** | 활성화 | 네트워크 운영에 필수 |

### 프로덕션을 위한 P2P 활성화

1. `docker-compose.yml`을 엽니다.
2. `--p2p.disable  # For local devnet only...`를 제거합니다.
3. P2P 구성 플래그를 다시 추가합니다.
   ```yaml
   --p2p.listen.ip=0.0.0.0
   --p2p.listen.tcp=9222
   --p2p.listen.udp=9222
   --p2p.advertise.ip=${P2P_ADVERTISE_IP}
   --p2p.advertise.tcp=9222
   --p2p.advertise.udp=9222
   --p2p.sequencer.key=${PRIVATE_KEY}
   ```
4. P2P 네트워킹 환경이 올바르게 구성되어 있는지 확인합니다.
   - `.env`의 `P2P_ADVERTISE_IP`를 공인 IP 주소로 설정합니다(127.0.0.1이 아님).
   - 포트 9222(TCP와 UDP 모두)가 인터넷에서 접근 가능한지 확인합니다.
   - P2P 트래픽을 허용하도록 적절한 방화벽 규칙을 구성합니다.
   - 더 나은 피어 탐색을 위해 bootnode 설정을 고려합니다.

```bash
# Example: Quick enable P2P for testing
sed -i '' '/--p2p.disable/d' docker-compose.yml
# Then add back P2P flags manually in docker-compose.yml
docker-compose restart op-node
```

## 문제 해결

### 일반적인 문제

1. **포트 충돌**: 포트 8545-8551과 9222를 사용할 수 있는지 확인하십시오.
2. **ETH 부족**: 배포 지갑에 충분한 Sepolia ETH가 있는지 확인하십시오.
3. **네트워크 타임아웃**: L1 RPC URL과 네트워크 연결 상태를 확인하십시오.
4. **Docker 문제**: Docker 데몬이 실행 중이고 리소스가 충분한지 확인하십시오.

### 배포 초기화

초기화하고 다시 배포하려면 다음을 실행합니다.

```bash
# Stop all services and clean up
make clean

# Re-run setup
make setup
make up
```

## 보안 참고 사항

- **개인 키를 절대 버전 관리 시스템에 커밋하지 마십시오**
- 프로덕션 배포에는 **하드웨어 보안 모듈(HSM)을 사용하십시오**
- Sepolia testnet에서 **gas 비용을 모니터링하십시오**
- **지갑 주소**와 배포 아티팩트를 **백업하십시오**

## 이 코드에 대하여

이 코드 예제는 Optimism 문서의 [L2 Rollup 생성 튜토리얼](/ko/chain-operators/tutorials/create-l2-rollup)과 함께 제공됩니다. 튜토리얼에서 다루는 개념을 보여 주는 완전한 동작 구현을 제공합니다.

## 기여

이 코드는 Optimism 문서의 일부입니다. 이슈 제보나 기여는 다음을 이용하십시오.

- **문서 이슈**: 메인 [Optimism 저장소](https://github.com/ethereum-optimism/optimism)에 보고하십시오.
- **코드 개선**: Optimism monorepo에 pull request를 제출하십시오.
- **튜토리얼 피드백**: 문서 피드백 기능을 이용하십시오.

## 라이선스

이 코드는 교육 및 테스트 목적으로 있는 그대로 제공됩니다. 라이선스 정보는 Optimism monorepo를 참조하십시오.
