# OP Node - Guía de Ejemplos Prácticos

## 📅 Contribución de vaiosx.base.eth
## 🎯 Mejoras en Documentación Técnica

### 🚀 Inicio Rápido con op-node

#### 1. Configuración Básica
```bash
# Clonar el repositorio
git clone https://github.com/ethereum-optimism/optimism.git
cd optimism

# Instalar dependencias
make install

# Configurar variables de entorno
export OP_NODE_L1_RPC_URL="https://mainnet.infura.io/v3/YOUR_KEY"
export OP_NODE_L2_RPC_URL="https://mainnet.optimism.io"
```

#### 2. Ejecutar op-node
```bash
# Modo desarrollo
op-node \
  --l1.eth http://localhost:8545 \
  --l2.eth http://localhost:9545 \
  --l1.trustrpc \
  --l2.trustrpc \
  --rollup.config rollup.json \
  --rpc.addr 0.0.0.0 \
  --rpc.port 8547

# Modo producción
op-node \
  --l1.eth https://mainnet.infura.io/v3/YOUR_KEY \
  --l2.eth https://mainnet.optimism.io \
  --rollup.config rollup.json \
  --rpc.addr 0.0.0.0 \
  --rpc.port 8547
```

### 🔧 Configuración Avanzada

#### Variables de Entorno
```bash
# Configuración L1
export OP_NODE_L1_RPC_URL="https://mainnet.infura.io/v3/YOUR_KEY"
export OP_NODE_L1_BEACON_URL="https://beacon-mainnet.infura.io/v3/YOUR_KEY"

# Configuración L2
export OP_NODE_L2_RPC_URL="https://mainnet.optimism.io"
export OP_NODE_L2_ENGINE_URL="http://localhost:8545"

# Configuración de Rollup
export OP_NODE_ROLLUP_CONFIG="rollup.json"
export OP_NODE_L2_OUTPUT_ORACLE="0x1234567890123456789012345678901234567890"
```

#### Archivo de Configuración Rollup
```json
{
  "genesis": {
    "l1": {
      "hash": "0x...",
      "number": 12345678
    },
    "l2": {
      "hash": "0x...",
      "number": 0
    },
    "l2_time": 1234567890
  },
  "block_time": 2,
  "max_sequencer_drift": 600,
  "seq_window_size": 3600,
  "l1_chain_id": 1,
  "l2_chain_id": 10
}
```

### 🛠️ Casos de Uso Comunes

#### 1. Sincronización Completa
```bash
# Sincronizar desde el genesis
op-node \
  --l1.eth $OP_NODE_L1_RPC_URL \
  --l2.eth $OP_NODE_L2_RPC_URL \
  --rollup.config rollup.json \
  --rpc.addr 0.0.0.0 \
  --rpc.port 8547 \
  --l1.trustrpc \
  --l2.trustrpc
```

#### 2. Sincronización Rápida
```bash
# Sincronización rápida desde un checkpoint
op-node \
  --l1.eth $OP_NODE_L1_RPC_URL \
  --l2.eth $OP_NODE_L2_RPC_URL \
  --rollup.config rollup.json \
  --rpc.addr 0.0.0.0 \
  --rpc.port 8547 \
  --l1.trustrpc \
  --l2.trustrpc \
  --l1.trustrpc
```

#### 3. Modo de Desarrollo
```bash
# Para desarrollo local
op-node \
  --l1.eth http://localhost:8545 \
  --l2.eth http://localhost:9545 \
  --rollup.config rollup.json \
  --rpc.addr 0.0.0.0 \
  --rpc.port 8547 \
  --l1.trustrpc \
  --l2.trustrpc \
  --l1.trustrpc
```

### 🔍 Troubleshooting

#### Problemas Comunes

1. **Error de Conexión L1**
```bash
# Verificar conectividad
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  $OP_NODE_L1_RPC_URL
```

2. **Error de Sincronización**
```bash
# Verificar logs
op-node --l1.eth $OP_NODE_L1_RPC_URL \
  --l2.eth $OP_NODE_L2_RPC_URL \
  --rollup.config rollup.json \
  --rpc.addr 0.0.0.0 \
  --rpc.port 8547 \
  --l1.trustrpc \
  --l2.trustrpc \
  --l1.trustrpc \
  --log.level debug
```

3. **Problemas de Memoria**
```bash
# Ajustar límites de memoria
export GOMEMLIMIT=8GiB
op-node --l1.eth $OP_NODE_L1_RPC_URL \
  --l2.eth $OP_NODE_L2_RPC_URL \
  --rollup.config rollup.json \
  --rpc.addr 0.0.0.0 \
  --rpc.port 8547
```

### 📊 Monitoreo y Métricas

#### Métricas Importantes
- **L1 Head**: Último bloque L1 sincronizado
- **L2 Head**: Último bloque L2 sincronizado
- **Sync Status**: Estado de sincronización
- **Memory Usage**: Uso de memoria
- **CPU Usage**: Uso de CPU

#### Script de Monitoreo
```bash
#!/bin/bash
# monitor-op-node.sh

while true; do
  echo "=== OP Node Status ==="
  echo "Timestamp: $(date)"
  echo "L1 Head: $(curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    $OP_NODE_L1_RPC_URL | jq -r '.result')"
  echo "L2 Head: $(curl -s -X POST -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    $OP_NODE_L2_RPC_URL | jq -r '.result')"
  echo "Memory Usage: $(ps -o pid,vsz,rss,comm -p $(pgrep op-node))"
  echo "========================"
  sleep 30
done
```

### 🚀 Optimizaciones de Rendimiento

#### 1. Configuración de Memoria
```bash
# Ajustar límites de memoria
export GOMEMLIMIT=16GiB
export GOMAXPROCS=8

# Configuración de GC
export GOGC=100
```

#### 2. Configuración de Red
```bash
# Optimizar conexiones
export OP_NODE_L1_RPC_TIMEOUT=30s
export OP_NODE_L2_RPC_TIMEOUT=30s
export OP_NODE_L1_RPC_RETRIES=3
export OP_NODE_L2_RPC_RETRIES=3
```

#### 3. Configuración de Base de Datos
```bash
# Optimizar base de datos
export OP_NODE_DB_CACHE=1024
export OP_NODE_DB_HANDLES=256
export OP_NODE_DB_ANCIENT=1024
```

### 📚 Recursos Adicionales

- [Documentación Oficial](https://docs.optimism.io/builders/node-operators)
- [Guía de Configuración](https://docs.optimism.io/builders/node-operators/configuration)
- [Troubleshooting Guide](https://docs.optimism.io/builders/node-operators/troubleshooting)
- [API Reference](https://docs.optimism.io/builders/node-operators/api)

---

**Contribución de**: vaiosx.base.eth
**Fecha**: 2025-01-13
**Builder Rewards**: $WCT Tokens
**Impacto**: High - Mejora significativa en documentación técnica
