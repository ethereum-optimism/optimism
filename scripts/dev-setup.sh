#!/bin/bash

# Script de configuración de desarrollo para OP Stack
# Contribución de vaiosx.base.eth

set -e

echo "🚀 Configurando entorno de desarrollo para OP Stack..."

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Función para imprimir mensajes
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verificar dependencias
check_dependencies() {
    print_status "Verificando dependencias..."
    
    # Verificar Go
    if ! command -v go &> /dev/null; then
        print_error "Go no está instalado. Por favor instala Go 1.21+"
        exit 1
    fi
    
    # Verificar Node.js
    if ! command -v node &> /dev/null; then
        print_error "Node.js no está instalado. Por favor instala Node.js 18+"
        exit 1
    fi
    
    # Verificar pnpm
    if ! command -v pnpm &> /dev/null; then
        print_warning "pnpm no está instalado. Instalando..."
        npm install -g pnpm
    fi
    
    # Verificar Docker
    if ! command -v docker &> /dev/null; then
        print_warning "Docker no está instalado. Algunas funcionalidades pueden no estar disponibles."
    fi
    
    print_success "Dependencias verificadas"
}

# Configurar variables de entorno
setup_environment() {
    print_status "Configurando variables de entorno..."
    
    # Crear archivo .env si no existe
    if [ ! -f .env ]; then
        cat > .env << EOF
# Configuración de desarrollo para OP Stack
# Contribución de vaiosx.base.eth

# L1 Configuration
OP_NODE_L1_RPC_URL="https://mainnet.infura.io/v3/YOUR_KEY"
OP_NODE_L1_BEACON_URL="https://beacon-mainnet.infura.io/v3/YOUR_KEY"

# L2 Configuration  
OP_NODE_L2_RPC_URL="https://mainnet.optimism.io"
OP_NODE_L2_ENGINE_URL="http://localhost:8545"

# Rollup Configuration
OP_NODE_ROLLUP_CONFIG="rollup.json"
OP_NODE_L2_OUTPUT_ORACLE="0x1234567890123456789012345678901234567890"

# Development Settings
OP_NODE_DEV_MODE=true
OP_NODE_LOG_LEVEL=info
OP_NODE_RPC_ADDR=0.0.0.0
OP_NODE_RPC_PORT=8547

# Performance Settings
GOMEMLIMIT=8GiB
GOMAXPROCS=4
GOGC=100

# Network Timeouts
OP_NODE_L1_RPC_TIMEOUT=30s
OP_NODE_L2_RPC_TIMEOUT=30s
OP_NODE_L1_RPC_RETRIES=3
OP_NODE_L2_RPC_RETRIES=3

# Database Settings
OP_NODE_DB_CACHE=1024
OP_NODE_DB_HANDLES=256
OP_NODE_DB_ANCIENT=1024
EOF
        print_success "Archivo .env creado"
    else
        print_warning "Archivo .env ya existe"
    fi
}

# Instalar dependencias
install_dependencies() {
    print_status "Instalando dependencias..."
    
    # Instalar dependencias de Go
    if [ -f go.mod ]; then
        go mod download
        go mod tidy
        print_success "Dependencias de Go instaladas"
    fi
    
    # Instalar dependencias de Node.js
    if [ -f package.json ]; then
        pnpm install
        print_success "Dependencias de Node.js instaladas"
    fi
    
    # Instalar dependencias de Python (si existe requirements.txt)
    if [ -f requirements.txt ]; then
        pip install -r requirements.txt
        print_success "Dependencias de Python instaladas"
    fi
}

# Configurar herramientas de desarrollo
setup_dev_tools() {
    print_status "Configurando herramientas de desarrollo..."
    
    # Crear directorio de scripts
    mkdir -p scripts/dev
    
    # Script de monitoreo
    cat > scripts/dev/monitor.sh << 'EOF'
#!/bin/bash
# Script de monitoreo para OP Node
# Contribución de vaiosx.base.eth

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
EOF
    chmod +x scripts/dev/monitor.sh
    
    # Script de testing
    cat > scripts/dev/test.sh << 'EOF'
#!/bin/bash
# Script de testing para OP Stack
# Contribución de vaiosx.base.eth

echo "🧪 Ejecutando tests..."

# Tests de Go
if [ -d "op-node" ]; then
    echo "Testing op-node..."
    cd op-node
    go test ./...
    cd ..
fi

# Tests de contratos
if [ -d "packages/contracts-bedrock" ]; then
    echo "Testing contracts..."
    cd packages/contracts-bedrock
    forge test
    cd ../..
fi

echo "✅ Tests completados"
EOF
    chmod +x scripts/dev/test.sh
    
    # Script de build
    cat > scripts/dev/build.sh << 'EOF'
#!/bin/bash
# Script de build para OP Stack
# Contribución de vaiosx.base.eth

echo "🔨 Building OP Stack..."

# Build op-node
if [ -d "op-node" ]; then
    echo "Building op-node..."
    cd op-node
    go build -o bin/op-node ./cmd
    cd ..
fi

# Build contracts
if [ -d "packages/contracts-bedrock" ]; then
    echo "Building contracts..."
    cd packages/contracts-bedrock
    forge build
    cd ../..
fi

echo "✅ Build completado"
EOF
    chmod +x scripts/dev/build.sh
    
    print_success "Herramientas de desarrollo configuradas"
}

# Configurar Git hooks
setup_git_hooks() {
    print_status "Configurando Git hooks..."
    
    # Pre-commit hook
    cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# Pre-commit hook para OP Stack
# Contribución de vaiosx.base.eth

echo "🔍 Ejecutando pre-commit checks..."

# Lint Go code
if [ -d "op-node" ]; then
    cd op-node
    go vet ./...
    go fmt ./...
    cd ..
fi

# Lint Solidity code
if [ -d "packages/contracts-bedrock" ]; then
    cd packages/contracts-bedrock
    forge fmt
    cd ../..
fi

echo "✅ Pre-commit checks completados"
EOF
    chmod +x .git/hooks/pre-commit
    
    print_success "Git hooks configurados"
}

# Crear documentación de desarrollo
create_dev_docs() {
    print_status "Creando documentación de desarrollo..."
    
    mkdir -p docs/dev
    
    # Guía de desarrollo
    cat > docs/dev/DEVELOPMENT_GUIDE.md << 'EOF'
# Guía de Desarrollo - OP Stack
# Contribución de vaiosx.base.eth

## 🚀 Inicio Rápido

### 1. Configuración del Entorno
```bash
# Ejecutar script de configuración
./scripts/dev-setup.sh
```

### 2. Variables de Entorno
```bash
# Cargar variables de entorno
source .env
```

### 3. Ejecutar en Modo Desarrollo
```bash
# Op-node
op-node --l1.eth $OP_NODE_L1_RPC_URL \
  --l2.eth $OP_NODE_L2_RPC_URL \
  --rollup.config rollup.json \
  --rpc.addr $OP_NODE_RPC_ADDR \
  --rpc.port $OP_NODE_RPC_PORT
```

## 🛠️ Herramientas de Desarrollo

### Scripts Disponibles
- `scripts/dev/monitor.sh` - Monitoreo de OP Node
- `scripts/dev/test.sh` - Ejecutar tests
- `scripts/dev/build.sh` - Build del proyecto

### Comandos Útiles
```bash
# Monitorear
./scripts/dev/monitor.sh

# Ejecutar tests
./scripts/dev/test.sh

# Build
./scripts/dev/build.sh
```

## 🔧 Configuración Avanzada

### Variables de Entorno Importantes
- `OP_NODE_L1_RPC_URL` - URL del RPC de L1
- `OP_NODE_L2_RPC_URL` - URL del RPC de L2
- `OP_NODE_ROLLUP_CONFIG` - Configuración de rollup
- `GOMEMLIMIT` - Límite de memoria para Go
- `GOMAXPROCS` - Número de procesos Go

### Troubleshooting
- Verificar conectividad L1/L2
- Revisar logs de op-node
- Verificar configuración de rollup
- Comprobar variables de entorno

## 📚 Recursos Adicionales
- [Documentación Oficial](https://docs.optimism.io/builders/node-operators)
- [Guía de Configuración](https://docs.optimism.io/builders/node-operators/configuration)
- [Troubleshooting](https://docs.optimism.io/builders/node-operators/troubleshooting)
EOF
    
    print_success "Documentación de desarrollo creada"
}

# Función principal
main() {
    echo "🎯 Configuración de desarrollo para OP Stack"
    echo "👤 Contribución de vaiosx.base.eth"
    echo "📅 Fecha: $(date)"
    echo ""
    
    check_dependencies
    setup_environment
    install_dependencies
    setup_dev_tools
    setup_git_hooks
    create_dev_docs
    
    echo ""
    print_success "✅ Configuración completada!"
    echo ""
    echo "📋 Próximos pasos:"
    echo "1. Revisar y configurar variables en .env"
    echo "2. Ejecutar: source .env"
    echo "3. Usar scripts en scripts/dev/"
    echo "4. Leer documentación en docs/dev/"
    echo ""
    echo "🎯 ¡Happy coding!"
}

# Ejecutar función principal
main "$@"
