module github.com/ethereum-optimism/optimism/go/proxyd

go 1.16

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/alicebob/miniredis v2.5.0+incompatible
	github.com/ethereum/go-ethereum v1.10.16
	github.com/go-redis/redis/v8 v8.11.4
	github.com/golang/snappy v0.0.4
	github.com/gomodule/redigo v1.8.8 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/prometheus/client_golang v1.11.0
	github.com/rs/cors v1.8.0
	github.com/stretchr/testify v1.7.0
	github.com/yuin/gopher-lua v0.0.0-20210529063254-f4c35e4016d9 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/docker/docker v1.4.2-0.20180625184442-8e610b2b55bf => github.com/docker/docker v1.6.1 // required to fix CVE-2015-3627

replace github.com/gin-gonic/gin v1.5.0 => github.com/gin-gonic/gin v1.6.3-0.20210406033725-bfc8ca285eb4 // indirect; required to fix CVE-2020-28483

replace github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2 // required to fix CVE-2021-3121

replace golang.org/x/text v0.3.6 => golang.org/x/text v0.3.7 // required to fix CVE-2021-38561
